package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/CrisH2307/repetition-study/internal/domain"
)

var _ domain.LLM = (*OpenRouter)(nil)

// OpenRouterPrefix marks a models.json id that goes through OpenRouter. It is
// stripped before the request, so "openrouter/anthropic/claude-sonnet-5.5" is
// sent as "anthropic/claude-sonnet-5.5". Bare claude-* and gemini-* ids stay
// on the direct vendor APIs.
const OpenRouterPrefix = "openrouter/"

const openRouterBase = "https://openrouter.ai/api/v1"

// OpenRouter fills any role over the chat completions API (OpenAI shape).
// Docs: https://openrouter.ai/docs/api-reference/overview (read 2026-11-14).
// Not run against the live API: every wire detail below comes from the docs.
type OpenRouter struct {
	key, base string
	http      *http.Client
	Model     string // a models.json id, with the openrouter/ prefix
	Retry     Retry
	Meter     *Meter
	// MaxTokens caps a streamed reply, JSONMaxTokens a JSON reply. On most
	// providers reasoning tokens count against the cap, so it also bounds them.
	MaxTokens, JSONMaxTokens int64
}

// NewOpenRouter builds a client. baseURL is empty for the real API.
func NewOpenRouter(apiKey, model, baseURL string) *OpenRouter {
	if baseURL == "" {
		baseURL = openRouterBase
	}
	return &OpenRouter{key: apiKey, base: strings.TrimRight(baseURL, "/"), http: &http.Client{},
		Model: model, MaxTokens: 200, JSONMaxTokens: 4096}
}

type orUsage struct {
	PromptTokens     int64 `json:"prompt_tokens"`
	CompletionTokens int64 `json:"completion_tokens"`
}

type orChunk struct {
	Choices []struct {
		Delta   struct{ Content string } `json:"delta"`
		Message struct{ Content string } `json:"message"`
	} `json:"choices"`
	Usage *orUsage `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Code    any    `json:"code"`
	} `json:"error"`
}

func (o *OpenRouter) body(req domain.LLMRequest, max int64) (map[string]any, string) {
	model := req.Model
	if model == "" {
		model = o.Model
	}
	msgs := []map[string]string{{"role": "system", "content": req.System}}
	for _, m := range req.Messages {
		role := "user"
		if m.Role == "assistant" {
			role = "assistant"
		}
		msgs = append(msgs, map[string]string{"role": role, "content": m.Content})
	}
	// Reasoning is kept minimal for both roles so replies stay short and cheap.
	// Reasoning tokens are billed as output and arrive inside completion_tokens,
	// so the meter counts them. UNVERIFIED on the live API: that every routed
	// model accepts effort "minimal" and folds reasoning into completion_tokens.
	return map[string]any{"model": strings.TrimPrefix(model, OpenRouterPrefix), "messages": msgs, "max_tokens": max,
		"reasoning": map[string]any{"effort": "minimal"}}, model
}

// post sends one request. A non-2xx status becomes an error that carries the
// status code, so retryable() sees 429 and 5xx.
func (o *OpenRouter) post(ctx context.Context, body map[string]any) (*http.Response, error) {
	b, _ := json.Marshal(body)
	r, err := http.NewRequestWithContext(ctx, http.MethodPost, o.base+"/chat/completions", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	r.Header.Set("Authorization", "Bearer "+o.key)
	r.Header.Set("Content-Type", "application/json")
	resp, err := o.http.Do(r)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode/100 != 2 {
		defer resp.Body.Close()
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("openrouter: status %d: %s", resp.StatusCode, msg)
	}
	return resp, nil
}

// JSON asks for schema-bound JSON (response_format json_schema) built from
// out's type. A decode failure is not retried.
func (o *OpenRouter) JSON(ctx context.Context, req domain.LLMRequest, out any) error {
	if err := o.Meter.check(); err != nil {
		return err
	}
	body, model := o.body(req, o.JSONMaxTokens)
	if s := schemaFor(out); s != nil {
		body["response_format"] = map[string]any{"type": "json_schema",
			"json_schema": map[string]any{"name": "reply", "strict": false, "schema": s}}
	} else {
		body["response_format"] = map[string]any{"type": "json_object"}
	}
	return o.Retry.do(ctx, func(actx context.Context) error {
		start := time.Now()
		resp, err := o.post(actx, body)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		var c orChunk
		if err := json.NewDecoder(resp.Body).Decode(&c); err != nil {
			return err
		}
		if c.Error != nil {
			return fmt.Errorf("openrouter: %v: %s", c.Error.Code, c.Error.Message)
		}
		var u orUsage
		if c.Usage != nil {
			u = *c.Usage
		}
		o.Meter.Record(Call{Model: model, In: u.PromptTokens, Out: u.CompletionTokens, Latency: time.Since(start)})
		if len(c.Choices) == 0 {
			return errNoRetry{errors.New("llm: openrouter reply has no choices")}
		}
		if err := json.Unmarshal([]byte(c.Choices[0].Message.Content), out); err != nil {
			return errNoRetry{fmt.Errorf("llm: reply is not the expected JSON: %w", err)}
		}
		return nil
	})
}

// Stream sends text deltas as they arrive. Usage arrives once, in the last
// chunk before [DONE]; a stream cut before that is metered at an upper bound.
func (o *OpenRouter) Stream(ctx context.Context, req domain.LLMRequest) (<-chan domain.Delta, error) {
	if err := o.Meter.check(); err != nil {
		return nil, err
	}
	body, model := o.body(req, o.MaxTokens)
	body["stream"] = true
	return streamRetry(ctx, o.Retry, func(actx context.Context, emit func(string) error) error {
		start := time.Now()
		resp, err := o.post(actx, body)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		var u orUsage
		answered, done := false, false
		defer func() {
			if answered && !done {
				in, out := cutShort(u.PromptTokens, u.CompletionTokens, req, o.MaxTokens)
				o.Meter.Record(Call{Model: model, In: in, Out: out, Latency: time.Since(start), CutShort: true})
			}
		}()
		sc := bufio.NewScanner(resp.Body)
		sc.Buffer(make([]byte, 64<<10), 1<<20)
		for sc.Scan() {
			line := sc.Text()
			// ": OPENROUTER PROCESSING" keep-alives start with a colon.
			data, ok := strings.CutPrefix(line, "data: ")
			if !ok {
				continue
			}
			if data == "[DONE]" {
				break
			}
			var c orChunk
			if err := json.Unmarshal([]byte(data), &c); err != nil {
				return errNoRetry{err}
			}
			answered = true
			if c.Error != nil {
				return errNoRetry{fmt.Errorf("openrouter: %v: %s", c.Error.Code, c.Error.Message)}
			}
			if c.Usage != nil {
				u = *c.Usage
			}
			if len(c.Choices) > 0 && c.Choices[0].Delta.Content != "" {
				if err := emit(c.Choices[0].Delta.Content); err != nil {
					return err
				}
			}
		}
		if err := sc.Err(); err != nil {
			return err
		}
		done = true
		o.Meter.Record(Call{Model: model, In: u.PromptTokens, Out: u.CompletionTokens, Latency: time.Since(start)})
		return nil
	}), nil
}
