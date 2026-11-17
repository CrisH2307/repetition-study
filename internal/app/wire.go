package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/CrisH2307/repetition-study/internal/auth"
	"github.com/CrisH2307/repetition-study/internal/checker"
	"github.com/CrisH2307/repetition-study/internal/course"
	"github.com/CrisH2307/repetition-study/internal/domain"
	"github.com/CrisH2307/repetition-study/internal/importer"
	"github.com/CrisH2307/repetition-study/internal/llm"
	"github.com/CrisH2307/repetition-study/internal/planner"
	"github.com/CrisH2307/repetition-study/internal/share"
	"github.com/CrisH2307/repetition-study/internal/srs"
	"github.com/CrisH2307/repetition-study/internal/store"
	"github.com/CrisH2307/repetition-study/internal/teach"
)

// DB is the part of *pgxpool.Pool (and of a pgx.Conn in tests) that every
// Postgres-backed piece needs.
type DB = planner.DB

// Config is the environment the server reads.
type Config struct {
	DatabaseURL  string
	StudentModel string // model id for the AI student, "fake" works offline
	CheckerModel string // model id for the checker, "fake" works offline, "fake-down" always fails
	Log          *slog.Logger
	// InsecureCookie sends the session cookie over plain http. Only for localhost dev.
	InsecureCookie bool
	// MonthlyCapUSD stops model checks for the rest of the calendar month once
	// the app has spent this much (D33). 0 means no cap.
	MonthlyCapUSD float64
}

type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now().UTC() }

// Wire builds the real scheduler, planner, checker, teach engine and store
// over one database and two model clients.
func Wire(db DB, student, checkerLLM domain.LLM, checkerModel string, clock domain.Clock, log *slog.Logger) Deps {
	sched := srs.New(db, clock)
	chk := &checker.Checker{LLM: checkerLLM, Model: checkerModel, Log: log}
	st := store.New(db)
	notes := course.Notes{Store: st, Now: clock.Now}
	eng := teach.New(userTeach{Store: st}, student, chk, sched, teach.Options{Log: log})
	return Deps{
		Auth:         &auth.Service{Store: st, Clock: clock},
		SecureCookie: true,
		Course:       st,
		Reviews:      st,
		Notes:        notes,
		Today:        planner.Today{},
		Own:          pgOwner{db: db, course: st},
		PlannerFor: func(userID string) domain.Planner {
			p := planner.New(db, clock)
			p.UserID = userID
			return p
		},
		Teach: eng,
		Sched: sched,
		// srs.Scheduler.Due drops the reason (a shadowed field), so use DueWithReasons.
		Due: func(ctx context.Context, now time.Time) ([]domain.DueItem, error) {
			rs, err := sched.DueWithReasons(ctx, now)
			if err != nil {
				return nil, err
			}
			out := make([]domain.DueItem, len(rs))
			for i, r := range rs {
				out[i] = r.DueItem
				out[i].Reason = r.Reason
			}
			return out, nil
		},
		Feedback: st,
		Privacy:  st,
		Imports:  st,
		Importer: &importer.Importer{Jobs: st, Courses: st, Review: st, Notes: notes, Now: clock.Now},
		Sharing:  &share.Service{Store: st, Users: st, Courses: st, Review: st, Notes: notes, Now: clock.Now},
		Planner:  planner.New(db, clock),
		Subjects: courseSubjects{st},
		Clock:    clock,
		Log:      log,
	}
}

// Build opens the pool, builds the model clients and returns the server.
func Build(ctx context.Context, cfg Config) (*http.Server, func(), error) {
	if cfg.DatabaseURL == "" {
		return nil, nil, errors.New("DATABASE_URL is not set")
	}
	if cfg.StudentModel == "" || cfg.CheckerModel == "" {
		return nil, nil, errors.New("STUDENT_MODEL and CHECKER_MODEL must be set (use fake for an offline run)")
	}
	if cfg.Log == nil {
		cfg.Log = slog.Default()
	}
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, nil, fmt.Errorf("database: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, nil, fmt.Errorf("database: %w", err)
	}
	meter := &llm.Meter{}
	student, err := llm.New(ctx, cfg.StudentModel, meter)
	if err != nil {
		pool.Close()
		return nil, nil, fmt.Errorf("student model: %w", err)
	}
	checkerLLM, err := llm.New(ctx, cfg.CheckerModel, meter)
	if err != nil {
		pool.Close()
		return nil, nil, fmt.Errorf("checker model: %w", err)
	}
	if cfg.MonthlyCapUSD > 0 {
		// Every model call is metered into the ledger; only the checker is stopped by the cap.
		rec := &spendRec{ledger: store.New(pool), total: func() float64 { _, _, c := meter.Total(); return c }, now: time.Now}
		student = metered{student, rec}
		checkerLLM = &capLLM{LLM: metered{checkerLLM, rec}, cap: cfg.MonthlyCapUSD, ledger: rec.ledger, rec: rec, now: time.Now}
	}
	if n, err := failInterrupted(ctx, pool, store.New(pool)); err != nil {
		pool.Close()
		return nil, nil, fmt.Errorf("uploads: %w", err)
	} else if n > 0 {
		cfg.Log.Warn("uploads left reading by a restart were marked interrupted", "count", n)
	}
	d := Wire(pool, student, checkerLLM, cfg.CheckerModel, systemClock{}, cfg.Log)
	d.SecureCookie = !cfg.InsecureCookie
	srv := &http.Server{
		Handler:           NewHandler(d),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second, // streams extend it per write, see handlers.turn
		IdleTimeout:       2 * time.Minute,
	}
	return srv, pool.Close, nil
}
