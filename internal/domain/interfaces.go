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

type Checker interface {
	Check(ctx context.Context, in CheckInput) ([]ClaimResult, error)
}

type Scheduler interface {
	ApplyTeachBack(ctx context.Context, conceptID string, results []ClaimResult) (ReviewEvent, error)
	ReviewCard(ctx context.Context, cardID string, grade Grade) (ReviewEvent, error)
	Override(ctx context.Context, conceptID string, dueAt time.Time, reason string) (ReviewEvent, error)
	Due(ctx context.Context, now time.Time) ([]DueItem, error)
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
