package planner

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/CrisH2307/repetition-study/internal/domain"
)

// DB is satisfied by *pgxpool.Pool and pgx.Tx, so tests can roll back.
type DB interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Begin(ctx context.Context) (pgx.Tx, error)
}

// Store implements domain.Planner on Postgres.
type Store struct {
	DB     DB
	Clock  domain.Clock
	Cfg    Config
	UserID string // FurtherPlanning has no user argument, so it reads this one
}

var _ domain.Planner = (*Store)(nil)

func New(db DB, clock domain.Clock) *Store {
	return &Store{DB: db, Clock: clock, Cfg: DefaultConfig(), UserID: domain.DemoUserID}
}

func (s *Store) FurtherPlanning(ctx context.Context, weekStart time.Time) (domain.FurtherPlanning, error) {
	r, err := s.Progress(ctx, weekStart)
	if err != nil {
		return domain.FurtherPlanning{}, err
	}
	out := domain.FurtherPlanning{WeekStart: r.WeekStart}
	for _, sub := range r.Subjects {
		sp := domain.SubjectProgress{SubjectID: sub.SubjectID, Name: sub.Name, NextWeekHours: sub.NextWeekHours}
		for _, c := range sub.Concepts {
			sp.Concepts = append(sp.Concepts, domain.ConceptProgress{
				ConceptID: c.ConceptID, Name: c.Name, Recall: c.Recall, Explanation: c.Explanation, NextDue: c.NextDue,
			})
		}
		out.Subjects = append(out.Subjects, sp)
	}
	return out, nil
}

// Generate replaces this user's unlocked, not yet started blocks for the week.
// Locked blocks and blocks already in the past stay put and count against weekly_hours.
func (s *Store) Generate(ctx context.Context, userID string, weekStart time.Time) (domain.Plan, error) {
	cfg, err := s.userConfig(ctx, userID)
	if err != nil {
		return domain.Plan{}, err
	}
	weekStart = cfg.LocalWeekStart(weekStart)
	now := s.Clock.Now().UTC()
	data, err := s.loadSubjects(ctx, userID)
	if err != nil {
		return domain.Plan{}, err
	}
	fixed, err := s.loadFixedBlocks(ctx, userID, weekStart, now)
	if err != nil {
		return domain.Plan{}, err
	}
	al, err := cfg.Allocate(now, weekStart, data, fixed)
	if err != nil {
		return domain.Plan{}, err
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return domain.Plan{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after commit
	if _, err := tx.Exec(ctx, `
		DELETE FROM plan_blocks p USING subjects su
		WHERE su.id = p.subject_id AND su.user_id = $1 AND NOT p.locked
		  AND p.starts_at >= $2 AND p.starts_at >= $3 AND p.starts_at < $4`,
		userID, now, weekStart, weekStart.AddDate(0, 0, 7)); err != nil {
		return domain.Plan{}, err
	}
	for i := range al.Blocks {
		b := &al.Blocks[i]
		if err := tx.QueryRow(ctx, `
			INSERT INTO plan_blocks (subject_id, concept_id, kind, starts_at, minutes, reason)
			VALUES ($1, $2, $3, $4, $5, $6) RETURNING id::text`,
			b.SubjectID, b.ConceptID, string(b.Kind), b.StartsAt, b.Minutes, b.Reason).Scan(&b.ID); err != nil {
			return domain.Plan{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Plan{}, err
	}

	blocks := append(fixed, al.Blocks...)
	sort.Slice(blocks, func(i, j int) bool {
		if !blocks[i].StartsAt.Equal(blocks[j].StartsAt) {
			return blocks[i].StartsAt.Before(blocks[j].StartsAt)
		}
		return blocks[i].ID < blocks[j].ID
	})
	return domain.Plan{WeekStart: weekStart, Blocks: blocks}, nil
}

// userConfig returns the store's Config with the user's time zone set.
// An unknown zone is an error: guessing UTC would put blocks at the wrong hours.
func (s *Store) userConfig(ctx context.Context, userID string) (Config, error) {
	var name string
	if err := s.DB.QueryRow(ctx, `SELECT time_zone FROM users WHERE id = $1`, userID).Scan(&name); err != nil {
		return Config{}, fmt.Errorf("planner: read time zone: %w", err)
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		return Config{}, fmt.Errorf("planner: user time zone %q: %w", name, err)
	}
	cfg := s.Cfg
	cfg.Loc = loc
	return cfg, nil
}

// loadFixedBlocks returns the week's blocks the allocator must not touch.
func (s *Store) loadFixedBlocks(ctx context.Context, userID string, weekStart, now time.Time) ([]domain.PlanBlock, error) {
	return s.queryBlocks(ctx, `
		SELECT p.id::text, p.subject_id::text, p.concept_id::text, p.kind, p.starts_at, p.minutes, p.reason, p.locked
		FROM plan_blocks p JOIN subjects su ON su.id = p.subject_id
		WHERE su.user_id = $1 AND p.starts_at >= $2 AND p.starts_at < $3 AND (p.locked OR p.starts_at < $4)
		ORDER BY p.starts_at, p.id`, userID, weekStart, weekStart.AddDate(0, 0, 7), now)
}

func (s *Store) queryBlocks(ctx context.Context, sql string, args ...any) ([]domain.PlanBlock, error) {
	rows, err := s.DB.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.PlanBlock
	for rows.Next() {
		var b domain.PlanBlock
		var kind string
		if err := rows.Scan(&b.ID, &b.SubjectID, &b.ConceptID, &kind, &b.StartsAt, &b.Minutes, &b.Reason, &b.Locked); err != nil {
			return nil, err
		}
		b.Kind, b.StartsAt = domain.PlanKind(kind), b.StartsAt.UTC()
		out = append(out, b)
	}
	return out, rows.Err()
}

// loadSubjects reads subjects, concepts, FSRS state and event counts for one user.
func (s *Store) loadSubjects(ctx context.Context, userID string) ([]SubjectData, error) {
	rows, err := s.DB.Query(ctx, `
		SELECT id::text, name, weekly_hours::float8, level, exam_date, weight::float8
		FROM subjects WHERE user_id = $1 ORDER BY id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var subs []SubjectData
	idx := map[string]int{}
	for rows.Next() {
		var sd SubjectData
		var lvl string
		if err := rows.Scan(&sd.Subject.ID, &sd.Subject.Name, &sd.Subject.WeeklyHours, &lvl, &sd.Subject.ExamDate, &sd.Subject.Weight); err != nil {
			return nil, err
		}
		sd.Subject.Level = domain.Level(lvl)
		if sd.Subject.ExamDate != nil {
			t := sd.Subject.ExamDate.UTC()
			sd.Subject.ExamDate = &t
		}
		idx[sd.Subject.ID] = len(subs)
		subs = append(subs, sd)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	crows, err := s.DB.Query(ctx, `
		SELECT c.subject_id::text, c.id::text, c.name, coalesce(cs.stability, 0), cs.last_review_at, coalesce(cs.state, 0), cs.due_at
		FROM concepts c JOIN subjects su ON su.id = c.subject_id
		LEFT JOIN concept_schedules cs ON cs.concept_id = c.id
		WHERE su.user_id = $1 ORDER BY c.id`, userID)
	if err != nil {
		return nil, err
	}
	defer crows.Close()
	for crows.Next() {
		var sid string
		var c ConceptData
		var last *time.Time
		var state int
		if err := crows.Scan(&sid, &c.ID, &c.Name, &c.Stability, &last, &state, &c.DueAt); err != nil {
			return nil, err
		}
		if last != nil {
			c.LastReview = last.UTC()
		}
		c.Scheduled = state != 0 && last != nil && c.Stability > 0
		subs[idx[sid]].Concepts = append(subs[idx[sid]].Concepts, c)
	}
	crows.Close()
	if err := crows.Err(); err != nil {
		return nil, err
	}

	erows, err := s.DB.Query(ctx, `
		SELECT c.subject_id::text, count(*) FROM review_events e
		JOIN concepts c ON c.id = e.concept_id JOIN subjects su ON su.id = c.subject_id
		WHERE su.user_id = $1 GROUP BY 1`, userID)
	if err != nil {
		return nil, err
	}
	defer erows.Close()
	for erows.Next() {
		var sid string
		var n int
		if err := erows.Scan(&sid, &n); err != nil {
			return nil, err
		}
		subs[idx[sid]].Events = n
	}
	if err := erows.Err(); err != nil {
		return nil, fmt.Errorf("planner: count events: %w", err)
	}
	return subs, nil
}
