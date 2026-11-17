package store

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/CrisH2307/repetition-study/internal/domain"
)

// None of these methods write review_events or concept_schedules (D30).

// SessionOwner is ErrNotFound unless the teach session's course is the user's.
func (s *Store) SessionOwner(ctx context.Context, userID, sessionID string) error {
	_, err := sessionConcept(ctx, s.pool, userID, sessionID)
	return err
}

func sessionConcept(ctx context.Context, db q, userID, sessionID string) (string, error) {
	var c string
	err := db.QueryRow(ctx, `SELECT ts.concept_id::text FROM teach_sessions ts
		JOIN concepts k ON k.id = ts.concept_id JOIN subjects s ON s.id = k.subject_id
		WHERE ts.id = $1 AND s.user_id = $2`, sessionID, userID).Scan(&c)
	return c, nf(err)
}

// claimInConcept is true when the live claim belongs to the concept.
func claimInConcept(ctx context.Context, db q, claimID, conceptID string) (bool, error) {
	var ok bool
	err := db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM claims WHERE id = $1 AND concept_id = $2)`, claimID, conceptID).Scan(&ok)
	if pgCode(err) == "22P02" {
		return false, nil
	}
	return ok, err
}

func (s *Store) SaveSelfGrades(ctx context.Context, userID, sessionID string, grades []domain.SelfGrade) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		concept, err := sessionConcept(ctx, tx, userID, sessionID)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `DELETE FROM self_grades WHERE session_id = $1`, sessionID); err != nil {
			return err
		}
		for _, g := range grades {
			if g.Rating != domain.RatingRight && g.Rating != domain.RatingNotSure && g.Rating != domain.RatingWrong {
				return &domain.UserError{Msg: "Pick right, not sure or wrong for each claim."}
			}
			if ok, err := claimInConcept(ctx, tx, g.ClaimID, concept); err != nil {
				return err
			} else if !ok {
				return &domain.UserError{Msg: "One of those claims is not part of this concept."}
			}
			if _, err := tx.Exec(ctx, `INSERT INTO self_grades (session_id, claim_id, user_id, rating) VALUES ($1, $2, $3, $4)
				ON CONFLICT (session_id, claim_id) DO UPDATE SET rating = $4`, sessionID, g.ClaimID, userID, g.Rating); err != nil {
				return err
			}
		}
		return nil
	})
}

const disputeCols = `id::text, session_id::text, claim_id::text, user_id::text, note, status, created_at`

func scanDispute(r pgx.Row) (d domain.Dispute, err error) {
	err = r.Scan(&d.ID, &d.SessionID, &d.ClaimID, &d.UserID, &d.Note, &d.Status, &d.CreatedAt)
	return
}

func (s *Store) CreateDispute(ctx context.Context, userID string, d domain.Dispute) (domain.Dispute, error) {
	concept, err := sessionConcept(ctx, s.pool, userID, d.SessionID)
	if err != nil {
		return domain.Dispute{}, err
	}
	if d.ClaimID != nil {
		if ok, err := claimInConcept(ctx, s.pool, *d.ClaimID, concept); err != nil {
			return domain.Dispute{}, err
		} else if !ok {
			return domain.Dispute{}, &domain.UserError{Msg: "That claim is not part of this session's concept."}
		}
	}
	return scanDispute(s.pool.QueryRow(ctx, `INSERT INTO disputes (session_id, claim_id, user_id, note) VALUES ($1, $2, $3, $4)
		RETURNING `+disputeCols, d.SessionID, d.ClaimID, userID, d.Note))
}

func (s *Store) AllDisputes(ctx context.Context) ([]domain.Dispute, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+disputeCols+` FROM disputes ORDER BY created_at, id`)
	return collect(rows, err, scanDispute)
}

// CourseView builds screen 6. Recall is the share of card reviews rated good
// or easy; explanation is the share of required live claims labelled supported
// in the concept's latest labelled session. Both nil without data.
func (s *Store) CourseView(ctx context.Context, userID, subjectID string) (domain.CourseView, error) {
	course, err := getCourse(ctx, s.pool, userID, subjectID)
	if err != nil {
		return domain.CourseView{}, err
	}
	v := domain.CourseView{Course: course}
	// The course is already checked as the user's, so list its uploads without a second lookup.
	rows, err := s.pool.Query(ctx, uploadSelect+`AND u.subject_id = $2 GROUP BY u.id ORDER BY u.created_at, u.id`, userID, subjectID)
	if v.Notes, err = collect(rows, err, scanUpload); err != nil {
		return v, err
	}

	rows, err = s.pool.Query(ctx, `
		WITH latest AS (
			SELECT DISTINCT ON (ts.concept_id) ts.id, ts.concept_id
			FROM teach_sessions ts JOIN concepts c ON c.id = ts.concept_id
			WHERE c.subject_id = $1 AND EXISTS (SELECT 1 FROM claim_labels x WHERE x.session_id = ts.id)
			ORDER BY ts.concept_id, ts.started_at DESC, ts.id DESC)
		SELECT c.id::text, c.name,
			(SELECT (count(*) FILTER (WHERE r.grade IN ('good', 'easy')))::float8 / nullif(count(*), 0)
			   FROM review_events r WHERE r.concept_id = c.id AND r.event_type = 'card_review'),
			CASE WHEN l.id IS NULL THEN NULL ELSE (
				SELECT (count(*) FILTER (WHERE EXISTS (SELECT 1 FROM claim_labels x
					WHERE x.session_id = l.id AND x.claim_id = k.id AND x.label = 'supported')))::float8 / nullif(count(*), 0)
				FROM claims k WHERE k.concept_id = c.id AND k.required AND k.deleted_at IS NULL) END,
			sch.due_at
		FROM concepts c LEFT JOIN latest l ON l.concept_id = c.id LEFT JOIN concept_schedules sch ON sch.concept_id = c.id
		WHERE c.subject_id = $1 ORDER BY c.name, c.id`, subjectID)
	if v.Cards, err = collect(rows, err, func(r pgx.Row) (c domain.StudyCard, err error) {
		err = r.Scan(&c.ConceptID, &c.Name, &c.Recall, &c.Explanation, &c.NextDue)
		return
	}); err != nil {
		return v, err
	}
	// One query for every card's live claims, not one per card.
	ids := make([]string, len(v.Cards))
	for i, c := range v.Cards {
		ids[i] = c.ConceptID
	}
	rows, err = s.pool.Query(ctx, `SELECT `+claimCols+` FROM claims c WHERE c.concept_id = ANY($1::uuid[]) AND c.deleted_at IS NULL
		ORDER BY c.created_at, c.id`, ids)
	claims, err := collect(rows, err, scanClaim)
	if err != nil {
		return v, err
	}
	at := make(map[string]int, len(v.Cards))
	for i, c := range v.Cards {
		at[c.ConceptID] = i
		v.Cards[i].Claims = []domain.Claim{}
	}
	for _, cl := range claims {
		v.Cards[at[cl.ConceptID]].Claims = append(v.Cards[at[cl.ConceptID]].Claims, cl)
	}

	// A mistake needs a quote, so labels without a verified citation are left out.
	// The passage is the block of the cited source that contains the quote.
	rows, err = s.pool.Query(ctx, `
		SELECT ts.id::text, ts.concept_id::text, cl.claim_id::text, cl.student_text, cl.label,
			cl.citation_quote, cl.citation_source_id::text, blk.page_id::text, coalesce(pg.number, blk.page), cl.created_at
		FROM claim_labels cl
		JOIN teach_sessions ts ON ts.id = cl.session_id JOIN concepts c ON c.id = ts.concept_id
		LEFT JOIN LATERAL (
			SELECT b.page_id, b.page FROM source_blocks b
			WHERE b.source_id = cl.citation_source_id
			  AND (position(cl.citation_quote in b.text_md) > 0 OR position(cl.citation_quote in b.text_norm) > 0)
			ORDER BY b.ordinal LIMIT 1) blk ON true
		LEFT JOIN pages pg ON pg.id = blk.page_id
		WHERE c.subject_id = $1 AND cl.label IN ('contradicted', 'partial')
		  AND cl.citation_source_id IS NOT NULL AND cl.citation_quote IS NOT NULL
		ORDER BY cl.created_at DESC, cl.id`, subjectID)
	v.Mistakes, err = collect(rows, err, func(r pgx.Row) (m domain.Mistake, err error) {
		err = r.Scan(&m.SessionID, &m.ConceptID, &m.ClaimID, &m.StudentText, &m.Label, &m.Quote, &m.SourceID, &m.PageID, &m.Page, &m.CreatedAt)
		return
	})
	return v, err
}

// StudyState loads the user's live courses, their concepts and the review
// events since since, oldest first.
func (s *Store) StudyState(ctx context.Context, userID string, since time.Time) (domain.StudyState, error) {
	st := domain.StudyState{Courses: []domain.CourseState{}}
	if err := s.pool.QueryRow(ctx, `SELECT id::text, name, time_zone FROM users WHERE id = $1`, userID).
		Scan(&st.User.ID, &st.User.Name, &st.User.TimeZone); err != nil {
		return st, nf(err)
	}
	subs, err := s.ListCourses(ctx, userID)
	if err != nil {
		return st, err
	}
	idx := map[string]int{}
	for i, sub := range subs {
		idx[sub.ID] = i
		st.Courses = append(st.Courses, domain.CourseState{Subject: sub, Concepts: []domain.ConceptState{}, Events: []domain.ReviewEvent{}})
	}

	rows, err := s.pool.Query(ctx, `
		SELECT c.subject_id::text, c.id::text, c.name, sch.due_at,
			EXISTS (SELECT 1 FROM cards d WHERE d.concept_id = c.id),
			EXISTS (SELECT 1 FROM review_events e WHERE e.concept_id = c.id AND e.event_type = 'teach_back')
		FROM concepts c JOIN subjects su ON su.id = c.subject_id LEFT JOIN concept_schedules sch ON sch.concept_id = c.id
		WHERE su.user_id = $1 AND su.archived_at IS NULL ORDER BY c.name, c.id`, userID)
	if err != nil {
		return st, err
	}
	type row struct {
		sub string
		c   domain.ConceptState
	}
	cs, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (x row, err error) {
		err = r.Scan(&x.sub, &x.c.ConceptID, &x.c.Name, &x.c.DueAt, &x.c.HasCards, &x.c.Taught)
		return
	})
	if err != nil {
		return st, err
	}
	for _, x := range cs {
		st.Courses[idx[x.sub]].Concepts = append(st.Courses[idx[x.sub]].Concepts, x.c)
	}

	rows, err = s.pool.Query(ctx, `
		SELECT c.subject_id::text, e.id::text, e.event_type, e.concept_id::text, e.card_id::text, e.grade, e.due_before, e.due_after,
			e.stability_before, e.stability_after, e.payload, e.created_at, e.session_id::text, e.claim_id::text, e.self_grade,
			e.checker_model, e.checker_version
		FROM review_events e JOIN concepts c ON c.id = e.concept_id JOIN subjects su ON su.id = c.subject_id
		WHERE su.user_id = $1 AND su.archived_at IS NULL AND e.created_at >= $2 ORDER BY e.created_at, e.id`, userID, since)
	if err != nil {
		return st, err
	}
	type ev struct {
		sub string
		e   domain.ReviewEvent
	}
	evs, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (x ev, err error) {
		e := &x.e
		err = r.Scan(&x.sub, &e.ID, &e.EventType, &e.ConceptID, &e.CardID, &e.Grade, &e.DueBefore, &e.DueAfter,
			&e.StabilityBefore, &e.StabilityAfter, &e.Payload, &e.CreatedAt, &e.SessionID, &e.ClaimID, &e.SelfGrade,
			&e.CheckerModel, &e.CheckerVersion)
		return
	})
	if err != nil {
		return st, err
	}
	for _, x := range evs {
		st.Courses[idx[x.sub]].Events = append(st.Courses[idx[x.sub]].Events, x.e)
	}
	return st, nil
}
