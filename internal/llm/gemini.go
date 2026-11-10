package llm

import (
	"context"
	"errors"
	"os"

	"google.golang.org/genai"

	"github.com/CrisH2307/repetition-study/internal/domain"
)

// Gemini streams the AI student's reply.
type Gemini struct {
	client *genai.Client
	Model  string
	Retry  Retry
	// MaxTokens caps the reply. The student speaks at most 2 short sentences.
	MaxTokens int32
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
	return &Gemini{client: c, Model: model, MaxTokens: 120}, nil
}

// NewGeminiFromEnv reads GEMINI_API_KEY and STUDENT_MODEL.
func NewGeminiFromEnv(ctx context.Context) (*Gemini, error) {
	key, model := os.Getenv("GEMINI_API_KEY"), os.Getenv("STUDENT_MODEL")
	if key == "" || model == "" {
		return nil, errors.New("llm: GEMINI_API_KEY and STUDENT_MODEL must be set")
	}
	return NewGemini(ctx, key, model, "")
}

func (g *Gemini) JSON(context.Context, domain.LLMRequest, any) error { return ErrUnsupported }

// Stream retries only until the first text arrives. After that a failure is
// sent as a terminal Delta.Err, because half a question cannot be replayed.
func (g *Gemini) Stream(ctx context.Context, req domain.LLMRequest) (<-chan domain.Delta, error) {
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
	cfg := &genai.GenerateContentConfig{
		SystemInstruction: genai.NewContentFromText(req.System, genai.RoleUser),
		MaxOutputTokens:   g.MaxTokens,
	}
	out := make(chan domain.Delta, 16)
	go func() {
		defer close(out)
		started := false
		err := g.Retry.do(ctx, func(actx context.Context) error {
			for resp, err := range g.client.Models.GenerateContentStream(actx, model, contents, cfg) {
				if err != nil {
					if started {
						return errNoRetry{err}
					}
					return err
				}
				if t := resp.Text(); t != "" {
					started = true
					select {
					case out <- domain.Delta{Text: t}:
					case <-ctx.Done():
						return ctx.Err()
					}
				}
			}
			return nil
		})
		if err != nil {
			var nr errNoRetry
			if errors.As(err, &nr) {
				err = nr.error
			}
			out <- domain.Delta{Err: err}
		}
	}()
	return out, nil
}

// errNoRetry marks an error that must not be retried.
type errNoRetry struct{ error }

func (e errNoRetry) Unwrap() error { return e.error }
