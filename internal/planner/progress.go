package planner

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/CrisH2307/repetition-study/internal/domain"
)

// Report is the full Further Planning data. domain.FurtherPlanning only has room
// for part of it (one recall number, no gaps, no change reasons), so Progress
// returns everything and FurtherPlanning maps what fits.
// Recall and explanation are never combined. nil means no data.
type Report struct {
	WeekStart time.Time
	Subjects  []SubjectReport
}

type SubjectReport struct {
	SubjectID          string
	Name               string
	Concepts           []ConceptReport
	MeanRetrievability *float64 // mean over concepts that have been scheduled, nil if none
	ThisWeekHours      float64
	NextWeekHours      float64
	Change             float64 // NextWeekHours - ThisWeekHours
	Reason             string
}

type ConceptReport struct {
	ConceptID      string
	Name           string
	Recall         *float64 // share of card reviews rated good or easy
	CardReviews    int
	Retrievability *float64 // FSRS retrievability now
	Explanation    *float64 // share of required claims supported in the latest teach-back
	Gaps           []Gap
	NextDue        *time.Time
}

type Gap struct {
	Claim      string
	Label      domain.ClaimLabel
	Citation   *domain.Citation // nil when the checker gave no verified citation
	Correction *string
}

func (s *Store) Progress(ctx context.Context, weekStart time.Time) (Report, error) {
	cfg, err := s.userConfig(ctx, s.UserID)
	if err != nil {
		return Report{}, err
	}
	weekStart = cfg.LocalWeekStart(weekStart)
	now := s.Clock.Now().UTC()
	data, err := s.loadSubjects(ctx, s.UserID)
	if err != nil {
		return Report{}, err
	}
	recall, err := s.loadRecall(ctx)
	if err != nil {
		return Report{}, err
	}
	expl, gaps, err := s.loadExplanation(ctx)
	if err != nil {
		return Report{}, err
	}

	// This week: stored blocks if any, else what the allocator would plan today.
	thisBlocks, err := s.queryBlocks(ctx, `
		SELECT p.id::text, p.subject_id::text, p.concept_id::text, p.kind, p.starts_at, p.minutes, p.reason, p.locked
		FROM plan_blocks p JOIN subjects su ON su.id = p.subject_id
		WHERE su.user_id = $1 AND p.starts_at >= $2 AND p.starts_at < $3`, s.UserID, weekStart, weekStart.AddDate(0, 0, 7))
	if err != nil {
		return Report{}, err
	}
	if len(thisBlocks) == 0 {
		a, err := cfg.Allocate(now, weekStart, data, nil)
		if err != nil {
			return Report{}, err
		}
		thisBlocks = a.Blocks
	}
	next := weekStart.AddDate(0, 0, 7)
	nextLocked, err := s.loadFixedBlocks(ctx, s.UserID, next, now)
	if err != nil {
		return Report{}, err
	}
	na, err := cfg.Allocate(now, next, data, nextLocked)
	if err != nil {
		return Report{}, err
	}
	nextBlocks := append(nextLocked, na.Blocks...)
	scores := map[string]SubjectScore{}
	for _, sc := range na.Scores {
		scores[sc.SubjectID] = sc
	}

	out := Report{WeekStart: weekStart}
	for _, sd := range data {
		id := sd.Subject.ID
		sr := SubjectReport{
			SubjectID: id, Name: sd.Subject.Name,
			ThisWeekHours: hours(thisBlocks, id), NextWeekHours: hours(nextBlocks, id),
		}
		sr.Change = sr.NextWeekHours - sr.ThisWeekHours
		var sum float64
		var scheduled, open int
		for _, c := range sd.Concepts {
			cr := ConceptReport{ConceptID: c.ID, Name: c.Name, Recall: recall[c.ID].share(), CardReviews: recall[c.ID].total,
				Explanation: expl[c.ID], Gaps: gaps[c.ID]}
			if c.Scheduled {
				r := retrievability(c, now)
				cr.Retrievability, cr.NextDue = &r, c.DueAt
				sum += r
				scheduled++
			}
			open += len(cr.Gaps)
			sr.Concepts = append(sr.Concepts, cr)
		}
		sort.Slice(sr.Concepts, func(i, j int) bool {
			if sr.Concepts[i].Name != sr.Concepts[j].Name {
				return sr.Concepts[i].Name < sr.Concepts[j].Name
			}
			return sr.Concepts[i].ConceptID < sr.Concepts[j].ConceptID
		})
		if scheduled > 0 {
			m := sum / float64(scheduled)
			sr.MeanRetrievability = &m
		}
		sr.Reason = s.changeReason(sr, scores[id], open)
		out.Subjects = append(out.Subjects, sr)
	}
	sort.Slice(out.Subjects, func(i, j int) bool {
		if out.Subjects[i].Name != out.Subjects[j].Name {
			return out.Subjects[i].Name < out.Subjects[j].Name
		}
		return out.Subjects[i].SubjectID < out.Subjects[j].SubjectID
	})
	return out, nil
}

func hours(blocks []domain.PlanBlock, subjectID string) float64 {
	m := 0
	for _, b := range blocks {
		if b.SubjectID == subjectID {
			m += b.Minutes
		}
	}
	return float64(m) / 60
}

func (s *Store) changeReason(sr SubjectReport, sc SubjectScore, openGaps int) string {
	head := "No change"
	if sr.Change != 0 {
		head = fmt.Sprintf("%+.1fh", sr.Change)
	}
	var why string
	if sc.Gap == 0 && sc.Score == 0 {
		why = "Projected recall at the exam meets the target, so no extra hours are needed."
	} else if sr.MeanRetrievability == nil {
		why = fmt.Sprintf("Nothing has been reviewed yet, so hours come from weekly hours and level (%.1fx).", sc.LevelNow)
	} else {
		why = fmt.Sprintf("Projected recall at the exam is %.0f%% against a %.0f%% target.", sc.Retention*100, s.Cfg.Target*100)
	}
	if sc.DaysToExam != nil && sc.Score > 0 {
		why += fmt.Sprintf(" The exam is %.0f days away.", *sc.DaysToExam)
	}
	if openGaps > 0 {
		why += fmt.Sprintf(" The latest teach-backs left %d open gaps.", openGaps)
	}
	return head + ": " + why
}

type recallCount struct{ good, total int }

func (r recallCount) share() *float64 {
	if r.total == 0 {
		return nil
	}
	v := float64(r.good) / float64(r.total)
	return &v
}

func (s *Store) loadRecall(ctx context.Context) (map[string]recallCount, error) {
	rows, err := s.DB.Query(ctx, `
		SELECT e.concept_id::text, count(*), count(*) FILTER (WHERE e.grade IN ('good', 'easy'))
		FROM review_events e JOIN concepts c ON c.id = e.concept_id JOIN subjects su ON su.id = c.subject_id
		WHERE su.user_id = $1 AND e.event_type = 'card_review' GROUP BY 1`, s.UserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]recallCount{}
	for rows.Next() {
		var id string
		var rc recallCount
		if err := rows.Scan(&id, &rc.total, &rc.good); err != nil {
			return nil, err
		}
		out[id] = rc
	}
	return out, rows.Err()
}

// loadExplanation reads the latest teach-back per concept: share of required
// claims labelled supported, and every non-supported label as a gap.
func (s *Store) loadExplanation(ctx context.Context) (map[string]*float64, map[string][]Gap, error) {
	claimRows, err := s.DB.Query(ctx, `
		SELECT cl.id::text, cl.concept_id::text, cl.text, cl.required
		FROM claims cl JOIN concepts c ON c.id = cl.concept_id JOIN subjects su ON su.id = c.subject_id
		WHERE su.user_id = $1`, s.UserID)
	if err != nil {
		return nil, nil, err
	}
	defer claimRows.Close()
	claimText := map[string]string{}
	required := map[string]map[string]bool{} // concept -> required claim ids
	for claimRows.Next() {
		var id, cid, text string
		var req bool
		if err := claimRows.Scan(&id, &cid, &text, &req); err != nil {
			return nil, nil, err
		}
		claimText[id] = text
		if req {
			if required[cid] == nil {
				required[cid] = map[string]bool{}
			}
			required[cid][id] = true
		}
	}
	claimRows.Close()
	if err := claimRows.Err(); err != nil {
		return nil, nil, err
	}

	rows, err := s.DB.Query(ctx, `
		WITH latest AS (
			SELECT DISTINCT ON (ts.concept_id) ts.id, ts.concept_id
			FROM teach_sessions ts JOIN concepts c ON c.id = ts.concept_id JOIN subjects su ON su.id = c.subject_id
			WHERE su.user_id = $1 AND EXISTS (SELECT 1 FROM claim_labels x WHERE x.session_id = ts.id)
			ORDER BY ts.concept_id, ts.started_at DESC, ts.id DESC)
		SELECT l.concept_id::text, cl.claim_id::text, cl.label, cl.student_text,
		       cl.citation_source_id::text, cl.citation_quote, cl.citation_locator, cl.correction
		FROM latest l JOIN claim_labels cl ON cl.session_id = l.id
		ORDER BY cl.created_at, cl.id`, s.UserID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	supported := map[string]map[string]bool{}
	seen := map[string]bool{}
	gaps := map[string][]Gap{}
	for rows.Next() {
		var cid, label, student string
		var claimID, srcID, quote, loc, corr *string
		if err := rows.Scan(&cid, &claimID, &label, &student, &srcID, &quote, &loc, &corr); err != nil {
			return nil, nil, err
		}
		seen[cid] = true
		if label == string(domain.LabelSupported) {
			if claimID != nil {
				if supported[cid] == nil {
					supported[cid] = map[string]bool{}
				}
				supported[cid][*claimID] = true
			}
			continue
		}
		g := Gap{Claim: student, Label: domain.ClaimLabel(label), Correction: corr}
		if claimID != nil && claimText[*claimID] != "" {
			g.Claim = claimText[*claimID]
		}
		if srcID != nil && quote != nil {
			g.Citation = &domain.Citation{SourceID: *srcID, Quote: *quote}
			if loc != nil {
				g.Citation.Locator = *loc
			}
		}
		gaps[cid] = append(gaps[cid], g)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	expl := map[string]*float64{}
	for cid := range seen {
		if len(required[cid]) == 0 {
			continue // nothing to measure against
		}
		n := 0
		for id := range required[cid] {
			if supported[cid][id] {
				n++
			}
		}
		v := float64(n) / float64(len(required[cid]))
		expl[cid] = &v
	}
	return expl, gaps, nil
}
