package domain

import (
	"context"
	"time"
)

type LLM interface {
	// Stream returns a channel of deltas. It closes when the reply ends.
	Stream(ctx context.Context, req LLMRequest) (<-chan Delta, error)
	// JSON asks for a JSON reply and decodes it into out.
	JSON(ctx context.Context, req LLMRequest, out any) error
}

type TeachEngine interface {
	Start(ctx context.Context, conceptID string) (TeachSession, error)
	// Turn streams the AI student's next question. It only asks, never teaches.
	Turn(ctx context.Context, sessionID, transcript, latex string) (<-chan Delta, error)
	Finish(ctx context.Context, sessionID string) (Report, error)
}

// ReportReader returns a finished session's report. Kept apart from
// TeachEngine so existing engines still compile; P0-OUTAGE implements it.
type ReportReader interface {
	GetReport(ctx context.Context, sessionID string) (Report, error)
}

// TeachStore is what the teach engine needs from persistence. Same method set
// as internal/teach.Store (checked in contract_test.go); P0-STORE implements it
// on Postgres.
type TeachStore interface {
	// LoadConcept returns the concept with its required claims and the
	// approved sources those claims cite.
	LoadConcept(ctx context.Context, conceptID string) (Concept, []Source, error)
	StartSession(ctx context.Context, conceptID string) (TeachSession, error)
	// SaveTurn stores the turn and its taught claim. The store assigns the claim ID.
	SaveTurn(ctx context.Context, t TaughtClaim, transcript, latex string) (TaughtClaim, error)
	SetAssistantText(ctx context.Context, sessionID string, turnNo int, text string) error
	// SaveResults replaces the session's claim labels.
	SaveResults(ctx context.Context, sessionID string, results []ClaimResult) error
	FinishSession(ctx context.Context, sessionID string) error
}

type Checker interface {
	Check(ctx context.Context, in CheckInput) ([]ClaimResult, error)
}

type Scheduler interface {
	ApplyTeachBack(ctx context.Context, conceptID string, results []ClaimResult) (ReviewEvent, error)
	ReviewCard(ctx context.Context, cardID string, grade Grade) (ReviewEvent, error)
	Override(ctx context.Context, conceptID string, dueAt time.Time, reason string) (ReviewEvent, error)
	Due(ctx context.Context, now time.Time) ([]DueItem, error)
}

// SessionScheduler applies a teach-back keyed by its session. Calling it again
// for the same session returns the first event and writes nothing new, so a
// retried Finish cannot double-schedule. Kept apart from Scheduler so existing
// schedulers still compile; P0-SRS implements it.
type SessionScheduler interface {
	ApplyTeachBackSession(ctx context.Context, sessionID, conceptID string, results []ClaimResult) (ReviewEvent, error)
}

// PackLoader writes a pack (subject, sources, blocks, concepts, claims,
// optional history) into storage for cmd/seed. Loading the same pack twice
// must not duplicate rows.
type PackLoader interface {
	LoadPack(ctx context.Context, p Pack) error
}

type Planner interface {
	Generate(ctx context.Context, userID string, weekStart time.Time) (Plan, error)
	FurtherPlanning(ctx context.Context, weekStart time.Time) (FurtherPlanning, error)
}

type Digester interface {
	Today(ctx context.Context, now time.Time) (Digest, error)
}

// Clock makes all date logic testable. It takes no context on purpose.
type Clock interface {
	Now() time.Time
}
