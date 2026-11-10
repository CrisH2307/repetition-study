package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/CrisH2307/repetition-study/internal/checker"
	"github.com/CrisH2307/repetition-study/internal/domain"
	"github.com/CrisH2307/repetition-study/internal/llm"
	"github.com/CrisH2307/repetition-study/internal/planner"
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
	CheckerModel string // model id for the checker, "fake" works offline
	Log          *slog.Logger
}

type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now().UTC() }

// Wire builds the real scheduler, planner, checker, teach engine and store
// over one database and two model clients.
func Wire(db DB, student, checkerLLM domain.LLM, checkerModel string, clock domain.Clock, log *slog.Logger) Deps {
	sched := srs.New(db, clock)
	chk := &checker.Checker{LLM: checkerLLM, Model: checkerModel, Log: log}
	eng := teach.New(store.New(db), student, chk, sched, teach.Options{Log: log})
	return Deps{
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
		Planner:  planner.New(db, clock),
		Subjects: pgSubjects{db},
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
	d := Wire(pool, student, checkerLLM, cfg.CheckerModel, systemClock{}, cfg.Log)
	srv := &http.Server{
		Handler:           NewHandler(d),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second, // streams extend it per write, see handlers.turn
		IdleTimeout:       2 * time.Minute,
	}
	return srv, pool.Close, nil
}
