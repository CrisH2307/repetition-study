// Package store is the Postgres implementation of domain.TeachStore and
// domain.PackLoader. All SQL is parameterised.
package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/CrisH2307/repetition-study/internal/domain"
)

// DB is the part of *pgxpool.Pool and *pgx.Conn the store uses. A pool is the
// production choice; importing pgxpool needs a go.mod change (puddle), which
// is outside this unit's boundary, so P0-INT adds it when it wires the pool.
type DB interface {
	Begin(ctx context.Context) (pgx.Tx, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type Store struct{ pool DB }

var (
	_ domain.TeachStore = (*Store)(nil)
	_ domain.PackLoader = (*Store)(nil)
)

func New(db DB) *Store { return &Store{pool: db} }

// LoadConcept returns the concept with its required claims, plus the course
// sources cited by any claim of the concept's subject. Allowlist sources are
// never returned (CLAUDE.md: approved sources are the course unit only).
func (s *Store) LoadConcept(ctx context.Context, id string) (domain.Concept, []domain.Source, error) {
	var c domain.Concept
	err := s.pool.QueryRow(ctx, `SELECT id, subject_id, name FROM concepts WHERE id = $1`, id).
		Scan(&c.ID, &c.SubjectID, &c.Name)
	if err != nil {
		return c, nil, fmt.Errorf("store: concept %q: %w", id, err)
	}
	rows, err := s.pool.Query(ctx, `SELECT id, concept_id, text, source_id, locator, required
		FROM claims WHERE concept_id = $1 AND required ORDER BY id`, id)
	if err != nil {
		return c, nil, err
	}
	if c.Claims, err = pgx.CollectRows(rows, func(r pgx.CollectableRow) (cl domain.Claim, err error) {
		err = r.Scan(&cl.ID, &cl.ConceptID, &cl.Text, &cl.SourceID, &cl.Locator, &cl.Required)
		return
	}); err != nil {
		return c, nil, err
	}
	rows, err = s.pool.Query(ctx, `SELECT id, kind, title, url, publisher, license, reviewed_by, reviewed_at, text
		FROM sources WHERE kind = 'course' AND id IN (
			SELECT cl.source_id FROM claims cl JOIN concepts co ON co.id = cl.concept_id WHERE co.subject_id = $1)
		ORDER BY id`, c.SubjectID)
	if err != nil {
		return c, nil, err
	}
	srcs, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (x domain.Source, err error) {
		err = r.Scan(&x.ID, &x.Kind, &x.Title, &x.URL, &x.Publisher, &x.License, &x.ReviewedBy, &x.ReviewedAt, &x.Text)
		return
	})
	return c, srcs, err
}

func (s *Store) StartSession(ctx context.Context, conceptID string) (domain.TeachSession, error) {
	ts := domain.TeachSession{ConceptID: conceptID}
	err := s.pool.QueryRow(ctx, `INSERT INTO teach_sessions (concept_id) VALUES ($1) RETURNING id`, conceptID).Scan(&ts.ID)
	return ts, err
}

// SaveTurn writes the turn and its taught claim in one transaction.
func (s *Store) SaveTurn(ctx context.Context, t domain.TaughtClaim, transcript, latex string) (domain.TaughtClaim, error) {
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `INSERT INTO teach_turns (session_id, turn_no, transcript, latex) VALUES ($1, $2, $3, $4)`,
			t.SessionID, t.TurnNo, transcript, latex); err != nil {
			return err
		}
		return tx.QueryRow(ctx, `INSERT INTO taught_claims (session_id, concept_id, student_text, turn_no)
			SELECT id, concept_id, $2, $3 FROM teach_sessions WHERE id = $1 RETURNING id, concept_id`,
			t.SessionID, t.StudentText, t.TurnNo).Scan(&t.ID, &t.ConceptID)
	})
	return t, err
}

func (s *Store) SetAssistantText(ctx context.Context, sessionID string, turnNo int, text string) error {
	return one(s.pool.Exec(ctx, `UPDATE teach_turns SET assistant_text = $3 WHERE session_id = $1 AND turn_no = $2`,
		sessionID, turnNo, text))
}

// SaveResults replaces the session's labels in one transaction.
func (s *Store) SaveResults(ctx context.Context, sessionID string, results []domain.ClaimResult) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `DELETE FROM claim_labels WHERE session_id = $1`, sessionID); err != nil {
			return err
		}
		for _, r := range results {
			var src, quote, loc *string
			if r.Citation != nil {
				src, quote, loc = &r.Citation.SourceID, &r.Citation.Quote, &r.Citation.Locator
			}
			if _, err := tx.Exec(ctx, `INSERT INTO claim_labels
				(session_id, claim_id, student_text, label, citation_source_id, citation_quote, citation_locator, correction, probe_hint)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
				sessionID, r.ClaimID, r.StudentText, r.Label, src, quote, loc, r.Correction, r.ProbeHint); err != nil {
				return err
			}
		}
		return nil
	})
}

// FinishSession is idempotent: a second call keeps the first finish time.
func (s *Store) FinishSession(ctx context.Context, sessionID string) error {
	return one(s.pool.Exec(ctx, `UPDATE teach_sessions SET status = 'finished', finished_at = COALESCE(finished_at, now())
		WHERE id = $1`, sessionID))
}

var errNoRow = errors.New("store: no matching row")

func one(tag interface{ RowsAffected() int64 }, err error) error {
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errNoRow
	}
	return nil
}

// LoadPack upserts a pack by id in one transaction, so loading twice adds no
// rows. History events are insert-only (review_events is append-only): an
// existing id is skipped, never updated.
func (s *Store) LoadPack(ctx context.Context, p domain.Pack) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		ex := func(sql string, args ...any) error { _, err := tx.Exec(ctx, sql, args...); return err }
		if p.Subject.ID == "" {
			return errors.New("store: pack subject has no id")
		}
		if err := ex(`INSERT INTO users (id, name) VALUES ($1, 'Demo Student') ON CONFLICT (id) DO NOTHING`, domain.DemoUserID); err != nil {
			return err
		}
		sub := p.Subject
		if err := ex(`INSERT INTO subjects (id, user_id, name, weekly_hours, level, exam_date, weight)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (id) DO UPDATE SET name = $3, weekly_hours = $4, level = $5, exam_date = $6, weight = $7`,
			sub.ID, domain.DemoUserID, sub.Name, sub.WeeklyHours, sub.Level, sub.ExamDate, sub.Weight); err != nil {
			return fmt.Errorf("subject: %w", err)
		}
		for _, x := range p.Sources {
			if err := ex(`INSERT INTO sources (id, kind, title, url, publisher, license, reviewed_by, reviewed_at, text, subject_id)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
				ON CONFLICT (id) DO UPDATE SET kind = $2, title = $3, url = $4, publisher = $5, license = $6,
					reviewed_by = $7, reviewed_at = $8, text = $9, subject_id = $10`,
				x.ID, x.Kind, x.Title, x.URL, x.Publisher, x.License, x.ReviewedBy, x.ReviewedAt, x.Text, sub.ID); err != nil {
				return fmt.Errorf("source %q: %w", x.ID, err)
			}
		}
		for _, b := range p.Blocks {
			if err := ex(`INSERT INTO source_blocks (id, source_id, page, ordinal, kind, text_md, text_norm, verified_by, verified_at)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
				ON CONFLICT (id) DO UPDATE SET source_id = $2, page = $3, ordinal = $4, kind = $5, text_md = $6,
					text_norm = $7, verified_by = $8, verified_at = $9`,
				b.ID, b.SourceID, b.Page, b.Ordinal, b.Kind, b.TextMD, b.TextNorm, b.VerifiedBy, b.VerifiedAt); err != nil {
				return fmt.Errorf("block %q: %w", b.ID, err)
			}
		}
		for _, c := range p.Concepts {
			if err := ex(`INSERT INTO concepts (id, subject_id, name) VALUES ($1, $2, $3)
				ON CONFLICT (id) DO UPDATE SET subject_id = $2, name = $3`, c.ID, sub.ID, c.Name); err != nil {
				return fmt.Errorf("concept %q: %w", c.ID, err)
			}
			for _, cl := range c.Claims {
				if err := ex(`INSERT INTO claims (id, concept_id, text, source_id, locator, required)
					VALUES ($1, $2, $3, $4, $5, $6)
					ON CONFLICT (id) DO UPDATE SET concept_id = $2, text = $3, source_id = $4, locator = $5, required = $6`,
					cl.ID, c.ID, cl.Text, cl.SourceID, cl.Locator, cl.Required); err != nil {
					return fmt.Errorf("claim %q: %w", cl.ID, err)
				}
			}
		}
		for _, h := range p.History {
			if h.ID == "" {
				return errors.New("store: history event has no id")
			}
			if len(h.Payload) == 0 {
				h.Payload = json.RawMessage(`{}`)
			}
			if err := ex(`INSERT INTO review_events (id, event_type, concept_id, card_id, grade, due_before, due_after,
					stability_before, stability_after, payload, created_at, session_id, claim_id, self_grade, checker_model, checker_version)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
				ON CONFLICT (id) DO NOTHING`,
				h.ID, h.EventType, h.ConceptID, h.CardID, h.Grade, h.DueBefore, h.DueAfter, h.StabilityBefore, h.StabilityAfter,
				h.Payload, h.CreatedAt, h.SessionID, h.ClaimID, h.SelfGrade, h.CheckerModel, h.CheckerVersion); err != nil {
				return fmt.Errorf("history %q: %w", h.ID, err)
			}
		}
		return nil
	})
}
