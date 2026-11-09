package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"google.golang.org/genai"

	"github.com/CrisH2307/repetition-study/internal/domain"
	"github.com/CrisH2307/repetition-study/internal/fakes"
)

type usage struct{ In, Out int64 }

// provider is one real or fake model that also reports token usage.
type provider interface {
	json(ctx context.Context, req domain.LLMRequest, out any) (usage, error)
}

type callStat struct {
	Latency time.Duration
	Usage   usage
}

// meter is a domain.LLM that records latency and tokens for every call.
type meter struct {
	p     provider
	mu    sync.Mutex
	calls []callStat
}

func (m *meter) JSON(ctx context.Context, req domain.LLMRequest, out any) error {
	start := time.Now()
	u, err := m.p.json(ctx, req, out)
	m.mu.Lock()
	m.calls = append(m.calls, callStat{time.Since(start), u})
	m.mu.Unlock()
	return err
}

func (m *meter) Stream(context.Context, domain.LLMRequest) (<-chan domain.Delta, error) {
	return nil, errors.New("evalcheck: streaming is not used")
}

// newProvider routes by model name prefix. "fake" replays a canned reply and
// needs no key. The Gemini and Claude paths are UNVERIFIED against the live
// APIs: no keys were available when this was written.
func newProvider(ctx context.Context, model string) (provider, error) {
	switch {
	case model == "fake":
		return fakeProvider{}, nil
	case strings.HasPrefix(model, "gemini"):
		key := os.Getenv("GEMINI_API_KEY")
		if key == "" {
			return nil, errors.New("GEMINI_API_KEY is not set")
		}
		c, err := genai.NewClient(ctx, &genai.ClientConfig{APIKey: key, Backend: genai.BackendGeminiAPI})
		if err != nil {
			return nil, err
		}
		return geminiProvider{c: c, model: model}, nil
	case strings.HasPrefix(model, "claude"):
		key := os.Getenv("ANTHROPIC_API_KEY")
		if key == "" {
			return nil, errors.New("ANTHROPIC_API_KEY is not set")
		}
		c := anthropic.NewClient(option.WithAPIKey(key))
		return claudeProvider{c: c, model: model}, nil
	}
	return nil, fmt.Errorf("unknown model %q: use gemini-*, claude-* or fake", model)
}

type fakeProvider struct{}

func (fakeProvider) json(ctx context.Context, req domain.LLMRequest, out any) (usage, error) {
	return usage{}, (&fakes.LLM{JSONReply: `{"results":[]}`}).JSON(ctx, req, out)
}

type geminiProvider struct {
	c     *genai.Client
	model string
}

func (g geminiProvider) json(ctx context.Context, req domain.LLMRequest, out any) (usage, error) {
	var contents []*genai.Content
	for _, m := range req.Messages {
		role := genai.RoleUser
		if m.Role == "assistant" {
			role = genai.RoleModel
		}
		contents = append(contents, genai.NewContentFromText(m.Content, genai.Role(role)))
	}
	cfg := &genai.GenerateContentConfig{
		SystemInstruction: genai.NewContentFromText(req.System, genai.RoleUser),
		ResponseMIMEType:  "application/json",
	}
	resp, err := g.c.Models.GenerateContent(ctx, g.model, contents, cfg)
	if err != nil {
		return usage{}, err
	}
	var u usage
	if md := resp.UsageMetadata; md != nil {
		// Thinking tokens are billed as output.
		u = usage{In: int64(md.PromptTokenCount), Out: int64(md.CandidatesTokenCount) + int64(md.ThoughtsTokenCount)}
	}
	return u, decode(resp.Text(), out)
}

type claudeProvider struct {
	c     anthropic.Client
	model string
}

func (cl claudeProvider) json(ctx context.Context, req domain.LLMRequest, out any) (usage, error) {
	var msgs []anthropic.MessageParam
	for _, m := range req.Messages {
		if m.Role == "assistant" {
			msgs = append(msgs, anthropic.NewAssistantMessage(anthropic.NewTextBlock(m.Content)))
		} else {
			msgs = append(msgs, anthropic.NewUserMessage(anthropic.NewTextBlock(m.Content)))
		}
	}
	msg, err := cl.c.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(cl.model),
		MaxTokens: 4096,
		System:    []anthropic.TextBlockParam{{Text: req.System}},
		Messages:  msgs,
	})
	if err != nil {
		return usage{}, err
	}
	u := usage{In: msg.Usage.InputTokens, Out: msg.Usage.OutputTokens}
	var sb strings.Builder
	for _, b := range msg.Content {
		if b.Type == "text" {
			sb.WriteString(b.AsText().Text)
		}
	}
	return u, decode(sb.String(), out)
}

// decode pulls the outermost JSON object out of text, so fenced or chatty
// replies still parse. Claude has no JSON mode here.
func decode(text string, out any) error {
	i, j := strings.Index(text, "{"), strings.LastIndex(text, "}")
	if i < 0 || j < i {
		return fmt.Errorf("no JSON object in reply: %.80q", text)
	}
	return json.Unmarshal([]byte(text[i:j+1]), out)
}
