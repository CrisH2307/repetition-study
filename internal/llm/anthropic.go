package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	"github.com/CrisH2307/repetition-study/internal/domain"
)

var _ domain.LLM = (*Anthropic)(nil)

// Anthropic fills any role: Stream and JSON both work.
type Anthropic struct {
	client anthropic.Client
	Model  string
	Retry  Retry
	Meter  *Meter
	// MaxTokens caps a JSON reply. StreamMaxTokens caps a streamed reply.
	MaxTokens, StreamMaxTokens int64
}

// NewAnthropic builds a client. baseURL is empty for the real API.
func NewAnthropic(apiKey, model, baseURL string) *Anthropic {
	// SDK retries are off: Retry owns retries so attempts are counted once.
	opts := []option.RequestOption{option.WithAPIKey(apiKey), option.WithMaxRetries(0)}
	if baseURL != "" {
		opts = append(opts, option.WithBaseURL(baseURL))
	}
	return &Anthropic{client: anthropic.NewClient(opts...), Model: model, MaxTokens: 4096, StreamMaxTokens: 200}
}

// NewAnthropicFromEnv reads ANTHROPIC_API_KEY and CHECKER_MODEL.
func NewAnthropicFromEnv() (*Anthropic, error) {
	key, model := os.Getenv("ANTHROPIC_API_KEY"), os.Getenv("CHECKER_MODEL")
	if key == "" || model == "" {
		return nil, errors.New("llm: ANTHROPIC_API_KEY and CHECKER_MODEL must be set")
	}
	return NewAnthropic(key, model, ""), nil
}

func (a *Anthropic) params(req domain.LLMRequest, max int64) (anthropic.MessageNewParams, string) {
	model := req.Model
	if model == "" {
		model = a.Model
	}
	msgs := make([]anthropic.MessageParam, 0, len(req.Messages))
	for _, m := range req.Messages {
		b := anthropic.NewTextBlock(m.Content)
		if m.Role == "assistant" {
			msgs = append(msgs, anthropic.NewAssistantMessage(b))
		} else {
			msgs = append(msgs, anthropic.NewUserMessage(b))
		}
	}
	return anthropic.MessageNewParams{
		Model:     anthropic.Model(model),
		MaxTokens: max,
		Messages:  msgs,
		System:    []anthropic.TextBlockParam{{Text: req.System}},
	}, model
}

// JSON uses structured output (output_config.format) built from out's type, so
// the reply is schema-bound JSON and nothing is scraped from prose. A decode
// failure is not retried. UNVERIFIED against the live API.
func (a *Anthropic) JSON(ctx context.Context, req domain.LLMRequest, out any) error {
	if err := a.Meter.check(); err != nil {
		return err
	}
	params, model := a.params(req, a.MaxTokens)
	if s := schemaFor(out); s != nil {
		params.OutputConfig = anthropic.OutputConfigParam{Format: anthropic.JSONOutputFormatParam{Schema: s}}
	} else {
		params.System[0].Text += "\n\nReply with one JSON value and nothing else."
	}
	return a.Retry.do(ctx, func(actx context.Context) error {
		start := time.Now()
		msg, err := a.client.Messages.New(actx, params)
		if err != nil {
			return err
		}
		a.Meter.Record(Call{Model: model, In: msg.Usage.InputTokens, Out: msg.Usage.OutputTokens, Latency: time.Since(start)})
		var sb strings.Builder
		for _, c := range msg.Content {
			if c.Type == "text" {
				sb.WriteString(c.AsText().Text)
			}
		}
		if err := json.Unmarshal([]byte(sb.String()), out); err != nil {
			return fmt.Errorf("%w: %w", errBadJSON, err)
		}
		return nil
	})
}

// Stream sends text deltas as they arrive.
func (a *Anthropic) Stream(ctx context.Context, req domain.LLMRequest) (<-chan domain.Delta, error) {
	if err := a.Meter.check(); err != nil {
		return nil, err
	}
	params, model := a.params(req, a.StreamMaxTokens)
	return streamRetry(ctx, a.Retry, func(actx context.Context, emit func(string) error) error {
		start := time.Now()
		stream := a.client.Messages.NewStreaming(actx, params)
		defer stream.Close()
		var acc anthropic.Message
		answered, done := false, false
		// Meter every attempt the vendor answered, also one cut short by a
		// cancel (the teach engine stops after 2 sentences) or an error.
		defer func() {
			if answered && !done {
				in, o := cutShort(acc.Usage.InputTokens, acc.Usage.OutputTokens, req, a.StreamMaxTokens)
				a.Meter.Record(Call{Model: model, In: in, Out: o, Latency: time.Since(start), CutShort: true})
			}
		}()
		for stream.Next() {
			answered = true
			ev := stream.Current()
			if err := acc.Accumulate(ev); err != nil {
				return errNoRetry{err}
			}
			if ev.Type == "content_block_delta" {
				if d := ev.AsContentBlockDelta(); d.Delta.Type == "text_delta" && d.Delta.Text != "" {
					if err := emit(d.Delta.Text); err != nil {
						return err
					}
				}
			}
		}
		if err := stream.Err(); err != nil {
			return err
		}
		done = true
		a.Meter.Record(Call{Model: model, In: acc.Usage.InputTokens, Out: acc.Usage.OutputTokens, Latency: time.Since(start)})
		return nil
	}), nil
}
