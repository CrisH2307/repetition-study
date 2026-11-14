package llm

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/CrisH2307/repetition-study/internal/domain"
	"github.com/CrisH2307/repetition-study/internal/fakes"
)

// New returns the client for a model id, routed by models.json (or the id
// prefix). The server and evalcheck both call this, so they run the same code.
// "fake" replays a canned checker reply and needs no key. "fake-down" needs no
// key either and fails every call, to rehearse a checker outage offline.
func New(ctx context.Context, model string, meter *Meter) (domain.LLM, error) {
	if model == "fake-down" {
		return down{}, nil
	}
	if model == "fake" {
		return &fake{meter: meter, inner: &fakes.LLM{JSONReply: `{"results":[]}`, Reply: "Why does that happen?"}}, nil
	}
	var m Models
	if meter != nil {
		m = meter.models()
	} else {
		m = DefaultModels()
	}
	v, err := m.Vendor(model)
	if err != nil {
		return nil, err
	}
	switch v {
	case "gemini":
		key := os.Getenv("GEMINI_API_KEY")
		if key == "" {
			return nil, fmt.Errorf("llm: GEMINI_API_KEY is not set")
		}
		g, err := NewGemini(ctx, key, model, "")
		if err != nil {
			return nil, err
		}
		g.Meter = meter
		return g, nil
	case "anthropic":
		key := os.Getenv("ANTHROPIC_API_KEY")
		if key == "" {
			return nil, fmt.Errorf("llm: ANTHROPIC_API_KEY is not set")
		}
		a := NewAnthropic(key, model, "")
		a.Meter = meter
		return a, nil
	}
	return nil, fmt.Errorf("llm: vendor %q for model %q is not supported", v, model)
}

// fake wraps fakes.LLM so offline runs are metered like real ones (0 tokens).
type fake struct {
	meter *Meter
	inner *fakes.LLM
}

func (f *fake) JSON(ctx context.Context, req domain.LLMRequest, out any) error {
	start := time.Now()
	err := f.inner.JSON(ctx, req, out)
	f.meter.Record(Call{Model: "fake", Latency: time.Since(start)})
	return err
}

func (f *fake) Stream(ctx context.Context, req domain.LLMRequest) (<-chan domain.Delta, error) {
	f.meter.Record(Call{Model: "fake"})
	return f.inner.Stream(ctx, req)
}

// down is a model that is always unreachable. Nothing is metered: no call is made.
type down struct{}

var errDown = errors.New("llm: fake-down model is always unavailable")

func (down) JSON(context.Context, domain.LLMRequest, any) error { return errDown }

func (down) Stream(context.Context, domain.LLMRequest) (<-chan domain.Delta, error) {
	return nil, errDown
}
