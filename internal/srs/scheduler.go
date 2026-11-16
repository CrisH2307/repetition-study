package srs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	fsrs "github.com/open-spaced-repetition/go-fsrs/v4"

	"github.com/CrisH2307/repetition-study/internal/domain"
)

var (
	_ domain.Scheduler        = (*Scheduler)(nil)
	_ domain.SessionScheduler = (*Scheduler)(nil)
)

// DB is satisfied by *pgxpool.Pool and *pgx.Conn. It is an interface only so this
// package does not import pgxpool, whose dependency is not in go.mod.
type DB interface {
	Begin(ctx context.Context) (pgx.Tx, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type Scheduler struct {
	db    DB
	clock domain.Clock
	fsrs  *fsrs.FSRS
}

func New(db DB, clock domain.Clock) *Scheduler {
	return &Scheduler{db: db, clock: clock, fsrs: newFSRS()}
}

type (
	eventIDKey   struct{}
	sessionIDKey struct{}
)

// WithEventID attaches a client-supplied event id (a uuid). The domain
// interface has no id parameter, so it rides on the context. A retry with the
// same id returns the stored event and writes nothing. Without one, the
// database generates the id and the write is not idempotent.
func WithEventID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, eventIDKey{}, id)
}

// run executes fn in a transaction holding a per-concept lock, after checking
// whether this event id was already written.
func (s *Scheduler) run(ctx context.Context, conceptID string, fn func(tx pgx.Tx) (domain.ReviewEvent, error)) (domain.ReviewEvent, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return domain.ReviewEvent{}, err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, conceptID); err != nil {
		return domain.ReviewEvent{}, err
	}
	if id, _ := ctx.Value(eventIDKey{}).(string); id != "" {
		ev, err := scanEvent(tx.QueryRow(ctx, eventSelect+` WHERE id = $1`, id))
		if err == nil {
			if ev.ConceptID != conceptID {
				return domain.ReviewEvent{}, fmt.Errorf("srs: event %s belongs to another concept", id)
			}
			return ev, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return domain.ReviewEvent{}, err
		}
	}
	if sid, _ := ctx.Value(sessionIDKey{}).(string); sid != "" {
		ev, err := scanEvent(tx.QueryRow(ctx, eventSelect+` WHERE session_id = $1 AND event_type = 'teach_back'`, sid))
		if err == nil {
			if ev.ConceptID != conceptID {
				return domain.ReviewEvent{}, fmt.Errorf("srs: session %s belongs to another concept", sid)
			}
			return ev, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return domain.ReviewEvent{}, err
		}
	}
	ev, err := fn(tx)
	if err != nil {
		return domain.ReviewEvent{}, err
	}
	return ev, tx.Commit(ctx)
}

const eventSelect = `SELECT id, event_type, concept_id, card_id, grade, due_before, due_after,
	stability_before, stability_after, payload, created_at, session_id FROM review_events`

func scanEvent(row pgx.Row) (domain.ReviewEvent, error) {
	var ev domain.ReviewEvent
	var et string
	var g *string
	var payload []byte
	err := row.Scan(&ev.ID, &et, &ev.ConceptID, &ev.CardID, &g, &ev.DueBefore, &ev.DueAfter,
		&ev.StabilityBefore, &ev.StabilityAfter, &payload, &ev.CreatedAt, &ev.SessionID)
	ev.EventType = domain.EventType(et)
	if g != nil {
		gg := domain.Grade(*g)
		ev.Grade = &gg
	}
	ev.Payload = payload
	ev.DueBefore, ev.DueAfter, ev.CreatedAt = ev.DueBefore.UTC(), ev.DueAfter.UTC(), ev.CreatedAt.UTC()
	return ev, err
}

// load returns the concept's FSRS card, or a new one when it has no schedule yet.
func (s *Scheduler) load(ctx context.Context, tx pgx.Tx, conceptID string, now time.Time) (fsrs.Card, error) {
	c := fsrs.NewCard(now)
	var last *time.Time
	var state int
	var reps, lapses int
	err := tx.QueryRow(ctx, `SELECT due_at, stability, difficulty, state, reps, lapses, last_review_at
		FROM concept_schedules WHERE concept_id = $1`, conceptID).
		Scan(&c.Due, &c.Stability, &c.Difficulty, &state, &reps, &lapses, &last)
	if errors.Is(err, pgx.ErrNoRows) {
		return c, nil
	}
	if err != nil {
		return c, err
	}
	c.State, c.Reps, c.Lapses = fsrs.State(state), uint64(reps), uint64(lapses)
	c.Due = c.Due.UTC()
	if last != nil {
		c.LastReview = last.UTC()
	}
	return c, nil
}

func save(ctx context.Context, tx pgx.Tx, conceptID string, c fsrs.Card) error {
	var last *time.Time
	if !c.LastReview.IsZero() {
		last = &c.LastReview
	}
	_, err := tx.Exec(ctx, `INSERT INTO concept_schedules (concept_id, due_at, stability, difficulty, state, reps, lapses, last_review_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT (concept_id) DO UPDATE SET due_at=$2, stability=$3, difficulty=$4, state=$5, reps=$6, lapses=$7, last_review_at=$8`,
		conceptID, c.Due, c.Stability, c.Difficulty, int(c.State), int(c.Reps), int(c.Lapses), last)
	return err
}

// record appends the event. It is the only writer of review_events.
func (s *Scheduler) record(ctx context.Context, tx pgx.Tx, et domain.EventType, conceptID string, cardID *string,
	g *domain.Grade, before, after fsrs.Card, payload any) (domain.ReviewEvent, error) {
	pj, err := json.Marshal(payload)
	if err != nil {
		return domain.ReviewEvent{}, err
	}
	id, _ := ctx.Value(eventIDKey{}).(string)
	sid, _ := ctx.Value(sessionIDKey{}).(string)
	var gs *string
	if g != nil {
		v := string(*g)
		gs = &v
	}
	return scanEvent(tx.QueryRow(ctx, `INSERT INTO review_events
		(id, event_type, concept_id, card_id, grade, due_before, due_after, stability_before, stability_after, payload, created_at, session_id)
		VALUES (COALESCE(NULLIF($1,'')::uuid, gen_random_uuid()), $2,$3,$4,$5,$6,$7,$8,$9,$10,$11, NULLIF($12,'')::uuid)
		RETURNING id, event_type, concept_id, card_id, grade, due_before, due_after, stability_before, stability_after, payload, created_at, session_id`,
		id, string(et), conceptID, cardID, gs, before.Due, after.Due, before.Stability, after.Stability, pj, s.clock.Now().UTC(), sid))
}

// review grades the concept, saves its schedule and appends the event.
func (s *Scheduler) review(ctx context.Context, tx pgx.Tx, et domain.EventType, conceptID string, cardID *string,
	g domain.Grade, payload any) (domain.ReviewEvent, error) {
	now := s.clock.Now().UTC()
	before, err := s.load(ctx, tx, conceptID, now)
	if err != nil {
		return domain.ReviewEvent{}, err
	}
	after, err := grade(s.fsrs, before, now, g)
	if err != nil {
		return domain.ReviewEvent{}, err
	}
	if err := save(ctx, tx, conceptID, after); err != nil {
		return domain.ReviewEvent{}, err
	}
	return s.record(ctx, tx, et, conceptID, cardID, &g, before, after, payload)
}

// ReviewCard grades one card. Cards carry no schedule, so the grade moves the
// concept's schedule and the event records which card was reviewed.
func (s *Scheduler) ReviewCard(ctx context.Context, cardID string, g domain.Grade) (domain.ReviewEvent, error) {
	var conceptID string
	if err := s.db.QueryRow(ctx, `SELECT concept_id FROM cards WHERE id = $1`, cardID).Scan(&conceptID); err != nil {
		return domain.ReviewEvent{}, fmt.Errorf("srs: card %s: %w", cardID, err)
	}
	return s.run(ctx, conceptID, func(tx pgx.Tx) (domain.ReviewEvent, error) {
		return s.review(ctx, tx, domain.EventCardReview, conceptID, &cardID, g, map[string]any{})
	})
}

// Override moves the due date only. Stability, difficulty and state are untouched.
func (s *Scheduler) Override(ctx context.Context, conceptID string, dueAt time.Time, reason string) (domain.ReviewEvent, error) {
	if reason == "" {
		return domain.ReviewEvent{}, errors.New("srs: override needs a reason")
	}
	return s.run(ctx, conceptID, func(tx pgx.Tx) (domain.ReviewEvent, error) {
		before, err := s.load(ctx, tx, conceptID, s.clock.Now().UTC())
		if err != nil {
			return domain.ReviewEvent{}, err
		}
		after := before
		after.Due = dueAt.UTC()
		if err := save(ctx, tx, conceptID, after); err != nil {
			return domain.ReviewEvent{}, err
		}
		return s.record(ctx, tx, domain.EventManualOverride, conceptID, nil, nil, before, after, map[string]string{"reason": reason})
	})
}

// ApplyTeachBack grades the worst required claim, moves the schedule and makes
// one gap card per contradicted, partial or absent result.
func (s *Scheduler) ApplyTeachBack(ctx context.Context, conceptID string, results []domain.ClaimResult) (domain.ReviewEvent, error) {
	return s.applyTeachBack(ctx, conceptID, results)
}

// ApplyTeachBackSession is ApplyTeachBack keyed by session id. A second call for
// the same session returns the first event and changes nothing. The per-concept
// lock serialises concurrent retries; the unique index
// review_events_teach_back_session_uniq is the backstop.
func (s *Scheduler) ApplyTeachBackSession(ctx context.Context, sessionID, conceptID string, results []domain.ClaimResult) (domain.ReviewEvent, error) {
	if sessionID == "" {
		return domain.ReviewEvent{}, errors.New("srs: teach-back needs a session id")
	}
	ctx = context.WithValue(ctx, sessionIDKey{}, sessionID)
	return s.applyTeachBack(ctx, conceptID, results)
}

func (s *Scheduler) applyTeachBack(ctx context.Context, conceptID string, results []domain.ClaimResult) (domain.ReviewEvent, error) {
	return s.run(ctx, conceptID, func(tx pgx.Tx) (domain.ReviewEvent, error) {
		claims, err := loadClaims(ctx, tx, conceptID)
		if err != nil {
			return domain.ReviewEvent{}, err
		}
		notRequired := map[string]bool{}
		for id, c := range claims {
			if !c.required {
				notRequired[id] = true
			}
		}
		g, err := teachBackGrade(results, notRequired)
		if err != nil {
			return domain.ReviewEvent{}, err
		}
		cardIDs, err := s.gapCards(ctx, tx, conceptID, claims, results)
		if err != nil {
			return domain.ReviewEvent{}, err
		}
		return s.review(ctx, tx, domain.EventTeachBack, conceptID, nil, g,
			map[string]any{"results": len(results), "gap_cards": cardIDs})
	})
}

type claimRow struct {
	text     string
	required bool
}

func loadClaims(ctx context.Context, tx pgx.Tx, conceptID string) (map[string]claimRow, error) {
	rows, err := tx.Query(ctx, `SELECT id, text, required FROM claims WHERE concept_id = $1`, conceptID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := map[string]claimRow{}
	for rows.Next() {
		var id string
		var c claimRow
		if err := rows.Scan(&id, &c.text, &c.required); err != nil {
			return nil, err
		}
		m[id] = c
	}
	return m, rows.Err()
}

// gapCards creates one editable card per contradicted, partial or absent result
// and returns the ids of the cards it created. A card already open for the same
// claim (or, for a result with no claim, the same front) is not duplicated.
// Every card has an answer from a correction (which carries its citation) or
// from the claim's own text, plus the citation when there is one. A result
// with neither a claim nor a correction has nothing to show, so it gets no card.
// ponytail: "open" means "exists"; cards have no status column.
func (s *Scheduler) gapCards(ctx context.Context, tx pgx.Tx, conceptID string, claims map[string]claimRow, results []domain.ClaimResult) ([]string, error) {
	var name string
	if err := tx.QueryRow(ctx, `SELECT name FROM concepts WHERE id = $1`, conceptID).Scan(&name); err != nil {
		return nil, err
	}
	created := []string{}
	for _, r := range results {
		if r.Label == domain.LabelSupported {
			continue
		}
		front, back, ok := cardText(name, claims, r)
		if !ok {
			continue
		}
		var exists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM cards WHERE concept_id = $1
			AND ((claim_id = $2::uuid) OR ($2::uuid IS NULL AND claim_id IS NULL AND front = $3)))`,
			conceptID, r.ClaimID, front).Scan(&exists); err != nil {
			return nil, err
		}
		if exists {
			continue
		}
		var id string
		if err := tx.QueryRow(ctx, `INSERT INTO cards (concept_id, claim_id, front, back, created_at, updated_at)
			VALUES ($1,$2,$3,$4,$5,$5) RETURNING id`, conceptID, r.ClaimID, front, back, s.clock.Now().UTC()).Scan(&id); err != nil {
			return nil, err
		}
		created = append(created, id)
	}
	return created, nil
}

func cardText(concept string, claims map[string]claimRow, r domain.ClaimResult) (front, back string, ok bool) {
	if r.StudentText != "" {
		front = fmt.Sprintf("%s: you said %q. What does the source say?", concept, r.StudentText)
	} else {
		front = fmt.Sprintf("%s: what did you leave out?", concept)
	}
	switch {
	case r.Correction != nil && *r.Correction != "":
		back = *r.Correction
	case r.ClaimID != nil && claims[*r.ClaimID].text != "":
		back = claims[*r.ClaimID].text
	default:
		return "", "", false
	}
	if r.Citation != nil {
		back += fmt.Sprintf("\n\nSource: %q (%s)", r.Citation.Quote, r.Citation.Locator)
	}
	return front, back, true
}

// ReasonedDue wraps DueItem for callers that use it by name. It must not declare
// its own Reason: that shadowed DueItem.Reason, so Due returned it empty.
type ReasonedDue struct{ domain.DueItem }

func (s *Scheduler) Due(ctx context.Context, now time.Time) ([]domain.DueItem, error) {
	r, err := s.DueWithReasons(ctx, now)
	if err != nil {
		return nil, err
	}
	out := make([]domain.DueItem, len(r))
	for i := range r {
		out[i] = r[i].DueItem
	}
	return out, nil
}

// DueWithReasons returns each due concept, then the cards under it, interleaved
// so a concept does not appear twice in a row while others are due.
// ponytail: the latest event is picked by created_at; two events in the same
// instant may tie. Fine for reasons, add a sequence column if it matters.
func (s *Scheduler) DueWithReasons(ctx context.Context, now time.Time) ([]ReasonedDue, error) {
	// Scheduled concepts due now come first, by due date. A concept with no
	// schedule row is new and due now, listed after them. Nothing is written.
	rows, err := s.db.Query(ctx, `SELECT id, name, due_at, event_type, grade, reason, bucket FROM (
		SELECT c.id, c.name, s.due_at, e.event_type, e.grade, e.payload->>'reason' AS reason, 0 AS bucket
		FROM concept_schedules s JOIN concepts c ON c.id = s.concept_id
		LEFT JOIN LATERAL (SELECT event_type, grade, payload FROM review_events
			WHERE concept_id = s.concept_id ORDER BY created_at DESC, id LIMIT 1) e ON true
		WHERE s.due_at <= $1
		UNION ALL
		SELECT c.id, c.name, $1::timestamptz, NULL, NULL, NULL, 1
		FROM concepts c WHERE NOT EXISTS (SELECT 1 FROM concept_schedules s WHERE s.concept_id = c.id)
	) d ORDER BY bucket, due_at, id`, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ReasonedDue
	var ids []string
	reasons := map[string]string{}
	for rows.Next() {
		var it ReasonedDue
		var et, g, why *string
		var bucket int
		if err := rows.Scan(&it.ConceptID, &it.ConceptName, &it.DueAt, &et, &g, &why, &bucket); err != nil {
			return nil, err
		}
		it.DueAt = it.DueAt.UTC()
		it.Reason = dueReason(et, g, why)
		if bucket == 1 {
			it.Reason = "New topic: not taught yet"
		}
		out, ids, reasons[it.ConceptID] = append(out, it), append(ids, it.ConceptID), it.Reason
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	crows, err := s.db.Query(ctx, `SELECT id, concept_id, claim_id, front, back FROM cards
		WHERE concept_id = ANY($1::uuid[]) ORDER BY created_at, id`, ids)
	if err != nil {
		return nil, err
	}
	defer crows.Close()
	for crows.Next() {
		var c domain.Card
		if err := crows.Scan(&c.ID, &c.ConceptID, &c.ClaimID, &c.Front, &c.Back); err != nil {
			return nil, err
		}
		var it ReasonedDue
		it.Reason = "card under due concept: " + reasons[c.ConceptID]
		it.ConceptID, it.CardID, it.Card = c.ConceptID, &c.ID, &c
		for _, p := range out {
			if p.ConceptID == c.ConceptID && p.CardID == nil {
				it.ConceptName, it.DueAt = p.ConceptName, p.DueAt
				break
			}
		}
		out = append(out, it)
	}
	if err := crows.Err(); err != nil {
		return nil, err
	}
	return interleave(out, func(r ReasonedDue) string { return r.ConceptID }), nil
}

func dueReason(et, g, why *string) string {
	switch {
	case et == nil:
		return "scheduled review"
	case *et == string(domain.EventManualOverride):
		if why != nil {
			return "due date set manually: " + *why
		}
		return "due date set manually"
	case *et == string(domain.EventTeachBack) && g != nil && (*g == string(domain.GradeAgain) || *g == string(domain.GradeHard)):
		return "teach-back found gaps (" + *g + ")"
	}
	return "scheduled review"
}
