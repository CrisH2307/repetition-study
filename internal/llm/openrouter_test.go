package llm

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/CrisH2307/repetition-study/internal/domain"
)

const orModel = "openrouter/google/gemini-3.8-flash"

var orReq = domain.LLMRequest{System: "s", Messages: []domain.Message{{Role: "user", Content: "hello there"}}}

func orServer(t *testing.T, got *string, reply func(w http.ResponseWriter)) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" || r.Header.Get("Authorization") != "Bearer k" {
			t.Errorf("bad request %s auth=%q", r.URL.Path, r.Header.Get("Authorization"))
		}
		b, _ := io.ReadAll(r.Body)
		*got = string(b)
		reply(w)
	}))
}

func TestOpenRouterStreamAndUsage(t *testing.T) {
	var body string
	srv := orServer(t, &body, func(w http.ResponseWriter) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, ": OPENROUTER PROCESSING\n\n")
		fmt.Fprint(w, `data: {"choices":[{"delta":{"content":"Why does"}}]}`+"\n\n")
		fmt.Fprint(w, `data: {"choices":[{"delta":{"content":" that happen?"}}]}`+"\n\n")
		fmt.Fprint(w, `data: {"choices":[{"delta":{"content":""},"finish_reason":"stop"}],"usage":{"prompt_tokens":11,"completion_tokens":5}}`+"\n\n")
		fmt.Fprint(w, "data: [DONE]\n\n")
	})
	defer srv.Close()
	o := NewOpenRouter("k", orModel, srv.URL)
	o.Meter = &Meter{}
	ch, err := o.Stream(context.Background(), orReq)
	if err != nil {
		t.Fatal(err)
	}
	if got := collect(t, ch); got != "Why does that happen?" {
		t.Fatalf("got %q", got)
	}
	if !strings.Contains(body, `"model":"google/gemini-3.8-flash"`) || !strings.Contains(body, `"stream":true`) {
		t.Errorf("prefix not stripped or stream missing: %s", body)
	}
	c := o.Meter.Calls()
	if len(c) != 1 || c[0].Model != orModel || c[0].In != 11 || c[0].Out != 5 || c[0].CutShort || c[0].Cost == 0 {
		t.Errorf("meter %+v", c)
	}
}

func TestOpenRouterJSON(t *testing.T) {
	var body string
	srv := orServer(t, &body, func(w http.ResponseWriter) {
		fmt.Fprint(w, `{"choices":[{"message":{"content":"{\"ok\": true}"}}],"usage":{"prompt_tokens":4,"completion_tokens":2}}`)
	})
	defer srv.Close()
	o := NewOpenRouter("k", "openrouter/anthropic/claude-sonnet-5.5", srv.URL)
	o.Meter = &Meter{}
	var out struct {
		OK bool `json:"ok"`
	}
	if err := o.JSON(context.Background(), orReq, &out); err != nil || !out.OK {
		t.Fatalf("err=%v out=%+v", err, out)
	}
	if !strings.Contains(body, `"json_schema"`) || !strings.Contains(body, `"model":"anthropic/claude-sonnet-5.5"`) {
		t.Errorf("request %s", body)
	}
	if c := o.Meter.Calls(); len(c) != 1 || c[0].In != 4 || c[0].Out != 2 {
		t.Errorf("meter %+v", c)
	}
}

func TestOpenRouterCanceledStreamIsMetered(t *testing.T) {
	srv := openStream(`data: {"choices":[{"delta":{"content":"Why does that. Happen. Again"}}]}` + "\n\n")
	defer srv.Close()
	o := NewOpenRouter("k", orModel, srv.URL)
	o.Meter = &Meter{}
	cutStream(t, o)
	c := o.Meter.Calls()
	// No usage arrived: input falls back to request bytes ("s" + "hello there"), output to the cap.
	if len(c) != 1 || !c[0].CutShort || c[0].In != 12 || c[0].Out != 200 {
		t.Fatalf("meter %+v", c)
	}
}

func TestOpenRouterSpendCapRefuses(t *testing.T) {
	var body string
	srv := orServer(t, &body, func(w http.ResponseWriter) { t.Error("request sent past the cap") })
	defer srv.Close()
	o := NewOpenRouter("k", orModel, srv.URL)
	o.Meter = &Meter{Cap: 1}
	o.Meter.Record(Call{Model: orModel, In: 1_000_000, Out: 1_000_000})
	var out struct{}
	if err := o.JSON(context.Background(), orReq, &out); !errors.Is(err, ErrBudget) {
		t.Fatalf("JSON err %v", err)
	}
	if _, err := o.Stream(context.Background(), orReq); !errors.Is(err, ErrBudget) {
		t.Fatalf("Stream err %v", err)
	}
}

func TestOpenRouterErrorStatusIsRetried(t *testing.T) {
	n := 0
	var body string
	srv := orServer(t, &body, func(w http.ResponseWriter) {
		if n++; n == 1 {
			http.Error(w, `{"error":{"code":503,"message":"down"}}`, 503)
			return
		}
		fmt.Fprint(w, `{"choices":[{"message":{"content":"{}"}}],"usage":{"prompt_tokens":1,"completion_tokens":1}}`)
	})
	defer srv.Close()
	o := NewOpenRouter("k", orModel, srv.URL)
	o.Retry = fast
	var out struct{}
	if err := o.JSON(context.Background(), orReq, &out); err != nil || n != 2 {
		t.Fatalf("err=%v hits=%d", err, n)
	}
}

func TestNewRoutesOpenRouter(t *testing.T) {
	ctx := context.Background()
	t.Setenv("OPENROUTER_API_KEY", "")
	if _, err := New(ctx, orModel, nil); err == nil || !strings.Contains(err.Error(), "OPENROUTER_API_KEY") {
		t.Fatalf("missing key err %v", err)
	}
	t.Setenv("OPENROUTER_API_KEY", "k")
	l, err := New(ctx, orModel, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := l.(*OpenRouter); !ok {
		t.Fatalf("got %T", l)
	}
	m := DefaultModels()
	for id, want := range map[string]string{
		"claude-sonnet-5-5":                      "anthropic",
		"gemini-3-flash":                         "gemini",
		"openrouter/anthropic/claude-sonnet-5.5": "openrouter",
		orModel:                                  "openrouter",
	} {
		if v, err := m.Vendor(id); err != nil || v != want {
			t.Errorf("%s routed to %q (%v), want %q", id, v, err, want)
		}
	}
	for _, id := range []string{"openrouter/anthropic/claude-sonnet-5.5", orModel} {
		if !m.Priced(id) {
			t.Errorf("%s has no price", id)
		}
	}
}
