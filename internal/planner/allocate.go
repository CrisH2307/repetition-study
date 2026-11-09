// Package planner builds the weekly plan and the Further Planning report.
// Allocate is pure: same input, same plan. Store (store.go) does the I/O.
package planner

import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/open-spaced-repetition/go-fsrs/v4"

	"github.com/CrisH2307/repetition-study/internal/domain"
)

// Config holds every tunable constant. All values are proposals, not measured.
type Config struct {
	Target         float64       // retention target R*, 0.90
	NewShare       float64       // share of all blocks reserved for new material, 0.30
	UrgencyCap     float64       // urgency = min(UrgencyCap, 1 + UrgencyDays/days_to_exam)
	UrgencyDays    float64       // 14
	LevelBeginner  float64       // 1.3
	LevelInter     float64       // 1.0
	LevelAdvanced  float64       // 0.8
	EvidenceEvents float64       // level fades out after this many review events, 20
	FullGap        float64       // a retention gap this big asks for the full weekly_hours, 0.5
	BlockMinutes   int           // 30
	DayStartHour   int           // first slot of each day, UTC hour, 9
	SlotsPerDay    int           // 8
	TeachLead      time.Duration // a teach block starts at least this long after generation, 24h
	MaxBlocks      int           // optional cap on blocks across all subjects, 0 = none
}

func DefaultConfig() Config {
	return Config{
		Target: 0.90, NewShare: 0.30, UrgencyCap: 3, UrgencyDays: 14,
		LevelBeginner: 1.3, LevelInter: 1.0, LevelAdvanced: 0.8, EvidenceEvents: 20,
		FullGap: 0.5, BlockMinutes: 30, DayStartHour: 9, SlotsPerDay: 8, TeachLead: 24 * time.Hour,
	}
}

type ConceptData struct {
	ID         string
	Name       string
	Stability  float64
	LastReview time.Time
	Scheduled  bool // false until the first review or teach-back
}

type SubjectData struct {
	Subject  domain.Subject
	Events   int // review_events across the subject's concepts
	Concepts []ConceptData
}

// SubjectScore exposes the score inputs so reports can explain a change.
type SubjectScore struct {
	SubjectID  string
	Score      float64
	Retention  float64 // mean projected retrievability at the horizon, 0 with no concepts
	Gap        float64 // max(0, target - retention)
	Urgency    float64
	LevelNow   float64
	DaysToExam *float64
	Demand     int // blocks the subject asks for this week, before locked blocks
	Blocks     int // new blocks assigned this week
}

type Allocation struct {
	Blocks []domain.PlanBlock // new blocks only, ascending by StartsAt, IDs unset
	Scores []SubjectScore     // ascending by subject id
}

var fsrsParams = fsrs.DefaultParam()

// retrievability at time at. New or never reviewed means 0.
func retrievability(c ConceptData, at time.Time) float64 {
	if !c.Scheduled || c.LastReview.IsZero() || c.Stability <= 0 {
		return 0
	}
	days := math.Max(0, at.Sub(c.LastReview).Hours()/24)
	return fsrsParams.ForgettingCurve(days, c.Stability)
}

func horizon(s domain.Subject, now, weekStart time.Time) time.Time {
	if s.ExamDate == nil {
		return weekStart.AddDate(0, 0, 7)
	}
	if s.ExamDate.Before(now) {
		return now
	}
	return *s.ExamDate
}

func (cfg Config) Score(s SubjectData, now, weekStart time.Time) SubjectScore {
	out := SubjectScore{SubjectID: s.Subject.ID, Urgency: 1}
	at := horizon(s.Subject, now, weekStart)
	if len(s.Concepts) > 0 {
		for _, c := range s.Concepts {
			out.Retention += retrievability(c, at)
		}
		out.Retention /= float64(len(s.Concepts))
	}
	out.Gap = math.Max(0, cfg.Target-out.Retention)
	if s.Subject.ExamDate != nil {
		d := s.Subject.ExamDate.Sub(now).Hours() / 24
		out.DaysToExam = &d
		out.Urgency = math.Min(cfg.UrgencyCap, 1+cfg.UrgencyDays/math.Max(d, 1))
	}
	base := map[domain.Level]float64{
		domain.LevelBeginner: cfg.LevelBeginner, domain.LevelIntermediate: cfg.LevelInter, domain.LevelAdvanced: cfg.LevelAdvanced,
	}[s.Subject.Level]
	if base == 0 {
		base = cfg.LevelInter
	}
	out.LevelNow = math.Pow(base, 1-math.Min(1, float64(s.Events)/cfg.EvidenceEvents))
	out.Score = s.Subject.Weight * out.LevelNow * out.Urgency * out.Gap
	return out
}

type want struct {
	kind    domain.PlanKind
	concept *ConceptData
	reason  string
}

// Allocate assigns 30 minute blocks greedily, then places them in free slots.
// locked blocks are never moved. They count against weekly_hours and block their slots.
func (cfg Config) Allocate(now, weekStart time.Time, in []SubjectData, locked []domain.PlanBlock) (Allocation, error) {
	subs := append([]SubjectData(nil), in...)
	sort.Slice(subs, func(i, j int) bool { return subs[i].Subject.ID < subs[j].Subject.ID })

	lockedBlocks := map[string]int{}
	lockedTeach := map[string]bool{}
	lockedNew := 0
	for _, b := range locked {
		lockedBlocks[b.SubjectID] += b.Minutes / cfg.BlockMinutes
		if b.Kind == domain.PlanTeach && b.ConceptID != nil {
			lockedTeach[*b.ConceptID] = true
		}
		if b.Kind == domain.PlanNew {
			lockedNew += b.Minutes / cfg.BlockMinutes
		}
	}

	// Score and demand. weekly_hours is a ceiling. Demand falls as the gap closes.
	scores := make([]SubjectScore, len(subs))
	room := make([]int, len(subs))
	for i, s := range subs {
		sc := cfg.Score(s, now, weekStart)
		ceiling := int(math.Floor(s.Subject.WeeklyHours*60/float64(cfg.BlockMinutes) + 1e-9))
		if sc.Score > 0 {
			sc.Demand = int(math.Ceil(float64(ceiling)*math.Min(1, sc.Gap/cfg.FullGap) - 1e-9))
		}
		scores[i] = sc
		room[i] = sc.Demand - lockedBlocks[s.Subject.ID]
	}

	// Greedy: highest score / (1 + blocks assigned), ties by subject id (subs is sorted).
	var seq []int
	assigned := make([]int, len(subs))
	for cfg.MaxBlocks == 0 || len(seq) < cfg.MaxBlocks {
		best, bestVal := -1, 0.0
		for i := range subs {
			if assigned[i] >= room[i] {
				continue
			}
			if v := scores[i].Score / float64(1+assigned[i]); best < 0 || v > bestVal {
				best, bestVal = i, v
			}
		}
		if best < 0 {
			break
		}
		assigned[best]++
		seq = append(seq, best)
	}
	for i := range scores {
		scores[i].Blocks = assigned[i]
	}

	// Which blocks are new material. Subjects without concepts are all new and count toward the share.
	noConcept := 0
	var eligible int
	for _, i := range seq {
		if len(subs[i].Concepts) == 0 {
			noConcept++
		} else {
			eligible++
		}
	}
	q := int(math.Round(cfg.NewShare*float64(len(seq)+len(locked)))) - lockedNew - noConcept
	q = max(0, min(q, eligible))

	// Per subject want lists: concepts weakest first, teach before review.
	order := make([][]ConceptData, len(subs))
	unscheduled := make([][]ConceptData, len(subs))
	plans := make([][]want, len(subs))
	for i, s := range subs {
		at := horizon(s.Subject, now, weekStart)
		cs := append([]ConceptData(nil), s.Concepts...)
		sort.Slice(cs, func(a, b int) bool {
			ra, rb := retrievability(cs[a], at), retrievability(cs[b], at)
			if ra != rb {
				return ra < rb
			}
			return cs[a].ID < cs[b].ID
		})
		order[i] = cs
		for _, c := range cs {
			if !c.Scheduled {
				unscheduled[i] = append(unscheduled[i], c)
			}
		}
		for p := 0; len(plans[i]) < assigned[i] && len(cs) > 0; p++ {
			c := cs[p%len(cs)]
			r := retrievability(c, at)
			if p < len(cs) && !lockedTeach[c.ID] {
				plans[i] = append(plans[i], want{domain.PlanTeach, &c, fmt.Sprintf(
					"Teach %q before reviewing it. Teach-backs are placed a day or more ahead so you study expecting to explain it.", c.Name)})
			}
			plans[i] = append(plans[i], want{domain.PlanReview, &c, fmt.Sprintf(
				"Review %q. Projected recall at %s is %.0f%% against a %.0f%% target.",
				c.Name, horizonLabel(s.Subject, at), r*100, cfg.Target*100)})
		}
	}

	// Walk the greedy sequence and decide each block's kind.
	var wants []want
	var subIdx []int
	newSeen := make([]int, len(subs))
	e := 0
	for _, i := range seq {
		s := subs[i]
		var w want
		isNew := len(s.Concepts) == 0
		if !isNew {
			isNew = (e+1)*q/eligible > e*q/eligible
			e++
		}
		switch {
		case len(s.Concepts) == 0:
			w = want{domain.PlanNew, nil, fmt.Sprintf(
				"No concepts yet for %s, so hours come from weekly hours (%.1f) and level (%s) alone.", s.Subject.Name, s.Subject.WeeklyHours, s.Subject.Level)}
		case isNew:
			w = want{domain.PlanNew, nil, fmt.Sprintf("New material. %.0f%% of hours are reserved for it.", cfg.NewShare*100)}
			if u := unscheduled[i]; len(u) > 0 {
				c := u[newSeen[i]%len(u)]
				w.concept = &c
				w.reason = fmt.Sprintf("New material: %q has not been taught yet. %.0f%% of hours are reserved for new material.", c.Name, cfg.NewShare*100)
			}
			newSeen[i]++
		case len(plans[i]) > 0:
			w, plans[i] = plans[i][0], plans[i][1:]
		default: // every teach was locked and nothing is left to plan
			w = want{domain.PlanNew, nil, fmt.Sprintf("New material. %.0f%% of hours are reserved for it.", cfg.NewShare*100)}
		}
		wants = append(wants, w)
		subIdx = append(subIdx, i)
	}

	blocks, err := cfg.place(now, weekStart, locked, wants, subIdx, subs)
	return Allocation{Blocks: blocks, Scores: scores}, err
}

func horizonLabel(s domain.Subject, at time.Time) string {
	if s.ExamDate == nil {
		return "the end of the week"
	}
	return "the exam on " + at.Format("2006-01-02")
}

// place spreads blocks over the week's free slots in order, so a teach block
// always precedes the review blocks of the same concept.
func (cfg Config) place(now, weekStart time.Time, locked []domain.PlanBlock, wants []want, subIdx []int, subs []SubjectData) ([]domain.PlanBlock, error) {
	slot := time.Duration(cfg.BlockMinutes) * time.Minute
	var free []time.Time
	for d := 0; d < 7; d++ {
		for j := 0; j < cfg.SlotsPerDay; j++ {
			t := weekStart.Add(time.Duration(d)*24*time.Hour + time.Duration(cfg.DayStartHour)*time.Hour + time.Duration(j)*slot)
			busy := t.Before(now)
			for _, b := range locked {
				if t.Before(b.StartsAt.Add(time.Duration(b.Minutes)*time.Minute)) && b.StartsAt.Before(t.Add(slot)) {
					busy = true
				}
			}
			if !busy {
				free = append(free, t)
			}
		}
	}
	out := make([]domain.PlanBlock, 0, len(wants))
	p := 0
	for k, w := range wants {
		idx := max(p, k*len(free)/len(wants))
		for idx < len(free) && w.kind == domain.PlanTeach && free[idx].Before(now.Add(cfg.TeachLead)) {
			idx++
		}
		if idx >= len(free) {
			return nil, fmt.Errorf("planner: no free slot left for block %d of %d in the week of %s", k+1, len(wants), weekStart.Format("2006-01-02"))
		}
		b := domain.PlanBlock{SubjectID: subs[subIdx[k]].Subject.ID, Kind: w.kind, StartsAt: free[idx], Minutes: cfg.BlockMinutes, Reason: w.reason}
		if w.concept != nil {
			id := w.concept.ID
			b.ConceptID = &id
		}
		out = append(out, b)
		p = idx + 1
	}
	return out, nil
}
