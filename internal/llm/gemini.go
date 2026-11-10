package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"google.golang.org/genai"

	"github.com/CrisH2307/repetition-study/internal/domain"
)

var _ domain.LLM = (*Gemini)(nil)

// Gemini fills any role: Stream and JSON both work.
type Gemini struct {
	client *genai.Client
	Model  string
	Retry  Retry
	Meter  *Meter
	// MaxTokens caps a streamed reply (the student speaks 2 short sentences,
	// about 200 tokens). JSONMaxTokens caps a JSON reply.
	MaxTokens, JSONMaxTokens int32
	// ThinkingBudget is always sent. 0 turns thinking off. Thinking tokens count
	// toward the cap and are billed as output, so an unset budget could eat the
	// whole cap (audit finding F10). UNVERIFIED on the real API.
	ThinkingBudget int32
}

// NewGemini builds a client. baseURL is empty for the real API (tests pass a stub server).
func NewGemini(ctx context.Context, apiKey, model, baseURL string) (*Gemini, error) {
	cfg := &genai.ClientConfig{APIKey: apiKey, Backend: genai.BackendGeminiAPI}
	if baseURL != "" {
		cfg.HTTPOptions = genai.HTTPOptions{BaseURL: baseURL}
	}
	c, err := genai.NewClient(ctx, cfg)
	if err != nil {
		return nil, err
	}
	g := &Gemini{client: c, Model: model, MaxTokens: 200, JSONMaxTokens: 8192}
	d := DefaultModels()
	g.ThinkingBudget = d.GeminiDefault.ThinkingBudget
	if b := d.Models[model].ThinkBudget; b != nil {
		g.ThinkingBudget = *b
	}
	return g, nil
}

// NewGeminiFromEnv reads GEMINI_API_KEY and STUDENT_MODEL.
func NewGeminiFromEnv(ctx context.Context) (*Gemini, error) {
	key, model := os.Getenv("GEMINI_API_KEY"), os.Getenv("STUDENT_MODEL")
	if key == "" || model == "" {
		return nil, errors.New("llm: GEMINI_API_KEY and STUDENT_MODEL must be set")
	}
	return NewGemini(ctx, key, model, "")
}

func (g *Gemini) request(req domain.LLMRequest, max int32) ([]*genai.Content, *genai.GenerateContentConfig, string) {
	model := req.Model
	if model == "" {
		model = g.Model
	}
	contents := make([]*genai.Content, 0, len(req.Messages))
	for _, m := range req.Messages {
		var role genai.Role = genai.RoleUser
		if m.Role == "assistant" {
			role = genai.RoleModel
		}
		contents = append(contents, genai.NewContentFromText(m.Content, role))
	}
	budget := g.ThinkingBudget
	cfg := &genai.GenerateContentConfig{
		SystemInstruction: genai.NewContentFromText(req.System, genai.RoleUser),
		MaxOutputTokens:   max,
		ThinkingConfig:    &genai.ThinkingConfig{ThinkingBudget: &budget},
	}
	return contents, cfg, model
}

// usage reads token counts. Thinking tokens are billed as output.
func usage(md *genai.GenerateContentResponseUsageMetadata) (in, out int64) {
	if md == nil {
		return 0, 0
	}
	return int64(md.PromptTokenCount), int64(md.CandidatesTokenCount) + int64(md.ThoughtsTokenCount)
}

// JSON asks for schema-bound JSON (responseJsonSchema built from out's type).
// A decode failure is not retried. UNVERIFIED against the live API.
func (g *Gemini) JSON(ctx context.Context, req domain.LLMRequest, out any) error {
	if err := g.Meter.check(); err != nil {
		return err
	}
	contents, cfg, model := g.request(req, g.JSONMaxTokens)
	cfg.ResponseMIMEType = "application/json"
	if s := schemaFor(out); s != nil {
		cfg.ResponseJsonSchema = s
	}
	return g.Retry.do(ctx, func(actx context.Context) error {
		start := time.Now()
		resp, err := g.client.Models.GenerateContent(actx, model, contents, cfg)
		if err != nil {
			return err
		}
		in, o := usage(resp.UsageMetadata)
		g.Meter.Record(Call{Model: model, In: in, Out: o, Latency: time.Since(start)})
		if err := json.Unmarshal([]byte(resp.Text()), out); err != nil {
			return errNoRetry{fmt.Errorf("llm: reply is not the expected JSON: %w", err)}
		}
		return nil
	})
}

// Stream sends text deltas as they arrive.
func (g *Gemini) Stream(ctx context.Context, req domain.LLMRequest) (<-chan domain.Delta, error) {
	if err := g.Meter.check(); err != nil {
		return nil, err
	}
	contents, cfg, model := g.request(req, g.MaxTokens)
	return streamRetry(ctx, g.Retry, func(actx context.Context, emit func(string) error) error {
		start := time.Now()
		var md *genai.GenerateContentResponseUsageMetadata
		for resp, err := range g.client.Models.GenerateContentStream(actx, model, contents, cfg) {
			if err != nil {
				return err
			}
			if resp.UsageMetadata != nil {
				md = resp.UsageMetadata
			}
			if t := resp.Text(); t != "" {
				if err := emit(t); err != nil {
					return err
				}
			}
		}
		in, o := usage(md)
		g.Meter.Record(Call{Model: model, In: in, Out: o, Latency: time.Since(start)})
		return nil
	}), nil
}
