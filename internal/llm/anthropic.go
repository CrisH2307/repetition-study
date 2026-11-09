package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	"github.com/CrisH2307/repetition-study/internal/domain"
)

// Anthropic returns JSON for the checker.
type Anthropic struct {
	client    anthropic.Client
	Model     string
	Retry     Retry
	MaxTokens int64
}

// NewAnthropic builds a client. baseURL is empty for the real API.
func NewAnthropic(apiKey, model, baseURL string) *Anthropic {
	// SDK retries are off: Retry owns retries so attempts are counted once.
	opts := []option.RequestOption{option.WithAPIKey(apiKey), option.WithMaxRetries(0)}
	if baseURL != "" {
		opts = append(opts, option.WithBaseURL(baseURL))
	}
	return &Anthropic{client: anthropic.NewClient(opts...), Model: model, MaxTokens: 2048}
}

// NewAnthropicFromEnv reads ANTHROPIC_API_KEY and CHECKER_MODEL.
func NewAnthropicFromEnv() (*Anthropic, error) {
	key, model := os.Getenv("ANTHROPIC_API_KEY"), os.Getenv("CHECKER_MODEL")
	if key == "" || model == "" {
		return nil, errors.New("llm: ANTHROPIC_API_KEY and CHECKER_MODEL must be set")
	}
	return NewAnthropic(key, model, ""), nil
}

func (a *Anthropic) Stream(context.Context, domain.LLMRequest) (<-chan domain.Delta, error) {
	return nil, ErrUnsupported
}

// JSON asks for JSON in the system prompt and decodes the first JSON value in
// the reply into out. A decode failure is not retried.
func (a *Anthropic) JSON(ctx context.Context, req domain.LLMRequest, out any) error {
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
	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(model),
		MaxTokens: a.MaxTokens,
		Messages:  msgs,
		System:    []anthropic.TextBlockParam{{Text: req.System + "\n\nReply with one JSON value and nothing else."}},
	}
	return a.Retry.do(ctx, func(actx context.Context) error {
		msg, err := a.client.Messages.New(actx, params)
		if err != nil {
			return err
		}
		var sb strings.Builder
		for _, c := range msg.Content {
			if c.Type == "text" {
				sb.WriteString(c.AsText().Text)
			}
		}
		raw := extractJSON(sb.String())
		if err := json.Unmarshal([]byte(raw), out); err != nil {
			return errNoRetry{fmt.Errorf("llm: reply is not the expected JSON: %w", err)}
		}
		return nil
	})
}

// extractJSON strips a markdown fence if the model added one.
func extractJSON(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```json")
		s = strings.TrimPrefix(s, "```")
		s = strings.TrimSuffix(strings.TrimSpace(s), "```")
	}
	return strings.TrimSpace(s)
}
