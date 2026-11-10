package llm

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/CrisH2307/repetition-study/internal/domain"
)

var fast = Retry{Attempts: 3, Backoff: time.Millisecond, Timeout: 2 * time.Second}

func TestRetryRetriesThenSucceeds(t *testing.T) {
	n := 0
	err := fast.do(context.Background(), func(context.Context) error {
		n++
		if n < 3 {
			return errors.New("got 503 from server")
		}
		return nil
	})
	if err != nil || n != 3 {
		t.Fatalf("err=%v attempts=%d", err, n)
	}
}

func TestRetrySkipsPermanentErrors(t *testing.T) {
	n := 0
	err := fast.do(context.Background(), func(context.Context) error { n++; return errors.New("400 bad request") })
	if err == nil || n != 1 {
		t.Fatalf("err=%v attempts=%d", err, n)
	}
}

func TestRetryTimesOutEachAttempt(t *testing.T) {
	r := Retry{Attempts: 2, Backoff: time.Millisecond, Timeout: 20 * time.Millisecond}
	n := 0
	err := r.do(context.Background(), func(ctx context.Context) error { n++; <-ctx.Done(); return ctx.Err() })
	if !errors.Is(err, context.DeadlineExceeded) || n != 2 {
		t.Fatalf("err=%v attempts=%d", err, n)
	}
}

// Anthropic against a stub server: request shape and fenced JSON decode.
func TestAnthropicJSON(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if hits.Add(1) == 1 {
			http.Error(w, `{"type":"error","error":{"type":"overloaded_error","message":"overloaded"}}`, 529)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"m1","type":"message","role":"assistant","model":"x","stop_reason":"end_turn",
		"content":[{"type":"text","text":"`+"```json\\n{\\\"ok\\\": true}\\n```"+`"}],"usage":{"input_tokens":1,"output_tokens":1}}`)
	}))
	defer srv.Close()
	a := NewAnthropic("k", "test-model", srv.URL)
	a.Retry = fast
	var out struct{ OK bool }
	err := a.JSON(context.Background(), domain.LLMRequest{System: "s", Messages: []domain.Message{{Role: "user", Content: "hi"}}}, &out)
	if err != nil || !out.OK || hits.Load() != 2 {
		t.Fatalf("err=%v out=%+v hits=%d", err, out, hits.Load())
	}
}

func TestGeminiStream(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if hits.Add(1) == 1 {
			http.Error(w, `{"error":{"code":503,"message":"unavailable","status":"UNAVAILABLE"}}`, 503)
			return
		}
		if !strings.Contains(r.URL.Path, "test-model") {
			t.Errorf("model missing from path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		for _, p := range []string{"Why does", " that happen?"} {
			fmt.Fprintf(w, "data: {\"candidates\":[{\"content\":{\"role\":\"model\",\"parts\":[{\"text\":%q}]}}]}\n\n", p)
			w.(http.Flusher).Flush()
		}
	}))
	defer srv.Close()
	g, err := NewGemini(context.Background(), "k", "test-model", srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	g.Retry = fast
	ch, err := g.Stream(context.Background(), domain.LLMRequest{System: "s", Messages: []domain.Message{{Role: "user", Content: "hi"}}})
	if err != nil {
		t.Fatal(err)
	}
	var got strings.Builder
	for d := range ch {
		if d.Err != nil {
			t.Fatal(d.Err)
		}
		got.WriteString(d.Text)
	}
	if got.String() != "Why does that happen?" {
		t.Fatalf("got %q (hits=%d)", got.String(), hits.Load())
	}
}

func TestUnsupportedMethods(t *testing.T) {
	g := &Gemini{}
	a := &Anthropic{}
	if !errors.Is(g.JSON(context.Background(), domain.LLMRequest{}, nil), ErrUnsupported) {
		t.Fatal("gemini JSON should be unsupported")
	}
	if _, err := a.Stream(context.Background(), domain.LLMRequest{}); !errors.Is(err, ErrUnsupported) {
		t.Fatal("anthropic Stream should be unsupported")
	}
}
