package main

import (
	"context"
	"fmt"
	"sync"

	"github.com/CrisH2307/repetition-study/internal/domain"
	"github.com/CrisH2307/repetition-study/internal/llm"
)

// budget is the spend cap for the whole run, across every suite and model.
type budget struct {
	meter *llm.Meter
	cap   float64 // USD, 0 means no cap
	hit   bool

	streams sync.WaitGroup // streams still running, whose cost is not recorded yet
}

// settle waits until every stream has ended and recorded its cost.
func (b *budget) settle() { b.streams.Wait() }

// reserve refuses a call whose worst-case cost could reach the cap.
func (b *budget) reserve(worst float64) error {
	b.settle()
	if b.cap <= 0 {
		return nil
	}
	if _, _, spent := b.meter.Total(); spent+worst >= b.cap {
		b.hit = true
		return fmt.Errorf("%w: $%.4f spent, the next call could cost up to $%.4f, cap $%.2f", errBudget, spent, worst, b.cap)
	}
	return nil
}

// overhead covers tokens the request does not show as text: the JSON schema,
// role markers. ponytail: a fixed guess, unverified against either vendor.
const overhead = 1000

// guard wraps a model client so no call starts unless its worst case fits
// under the cap. Worst case: every byte sent is one input token (a token is
// never shorter than one byte in either vendor's tokenizer), the client's
// whole output cap is used, and every retry attempt is billed.
type guard struct {
	inner              domain.LLM
	model              string
	jsonOut, streamOut int64
	attempts           int
	bud                *budget
}

func newGuard(l domain.LLM, model string, bud *budget) *guard {
	g := &guard{inner: l, model: model, bud: bud}
	switch c := l.(type) {
	case *llm.Anthropic:
		g.jsonOut, g.streamOut, g.attempts = c.MaxTokens, c.StreamMaxTokens, c.Retry.Attempts
	case *llm.Gemini:
		// Thinking tokens are billed as output. Whether they sit inside
		// MaxOutputTokens is unverified, so they are added on top.
		think := int64(max(c.ThinkingBudget, 0))
		g.jsonOut, g.streamOut, g.attempts = int64(c.JSONMaxTokens)+think, int64(c.MaxTokens)+think, c.Retry.Attempts
	}
	if g.attempts <= 0 {
		g.attempts = 3 // llm.Retry's default
	}
	return g
}

func (g *guard) worst(req domain.LLMRequest, out int64) float64 {
	in := int64(len(req.System) + overhead)
	for _, m := range req.Messages {
		in += int64(len(m.Content))
	}
	return float64(g.attempts) * g.bud.meter.Models.Cost(g.model, in, out)
}

func (g *guard) JSON(ctx context.Context, req domain.LLMRequest, out any) error {
	if err := g.bud.reserve(g.worst(req, g.jsonOut)); err != nil {
		return err
	}
	return g.inner.JSON(ctx, req, out)
}

// Stream forwards the deltas and tells the budget when the stream has ended,
// so the next call sees this one's cost.
func (g *guard) Stream(ctx context.Context, req domain.LLMRequest) (<-chan domain.Delta, error) {
	if err := g.bud.reserve(g.worst(req, g.streamOut)); err != nil {
		return nil, err
	}
	up, err := g.inner.Stream(ctx, req)
	if err != nil || up == nil {
		return up, err
	}
	out := make(chan domain.Delta)
	g.bud.streams.Add(1)
	go func() {
		defer g.bud.streams.Done()
		defer close(out)
		for d := range up { // read to the end even if the caller stopped
			select {
			case out <- d:
			case <-ctx.Done():
			}
		}
	}()
	return out, nil
}
