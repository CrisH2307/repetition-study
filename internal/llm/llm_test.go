package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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

// Anthropic against a stub server: retry, structured-output request and usage metering.
func TestAnthropicJSON(t *testing.T) {
	var hits atomic.Int32
	var body string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		body = string(b)
		if hits.Add(1) == 1 {
			http.Error(w, `{"type":"error","error":{"type":"overloaded_error","message":"overloaded"}}`, 529)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"m1","type":"message","role":"assistant","model":"x","stop_reason":"end_turn",
		"content":[{"type":"text","text":"{\"ok\": true}"}],"usage":{"input_tokens":1,"output_tokens":1}}`)
	}))
	defer srv.Close()
	a := NewAnthropic("k", "test-model", srv.URL)
	a.Retry = fast
	a.Meter = &Meter{}
	var out struct {
		OK bool `json:"ok"`
	}
	err := a.JSON(context.Background(), domain.LLMRequest{System: "s", Messages: []domain.Message{{Role: "user", Content: "hi"}}}, &out)
	if err != nil || !out.OK || hits.Load() != 2 {
		t.Fatalf("err=%v out=%+v hits=%d", err, out, hits.Load())
	}
	if !strings.Contains(body, `"output_config"`) || !strings.Contains(body, `"json_schema"`) {
		t.Errorf("structured output not requested: %s", body)
	}
	if c := a.Meter.Calls(); len(c) != 1 || c[0].In != 1 || c[0].Out != 1 {
		t.Errorf("meter %+v", c)
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

func collect(t *testing.T, ch <-chan domain.Delta) string {
	t.Helper()
	var b strings.Builder
	for d := range ch {
		if d.Err != nil {
			t.Fatal(d.Err)
		}
		b.WriteString(d.Text)
	}
	return b.String()
}

// Anthropic can stream (the student role) and the meter sees the usage.
func TestAnthropicStream(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		for _, e := range []string{
			`{"type":"message_start","message":{"id":"m","type":"message","role":"assistant","model":"x","content":[],"usage":{"input_tokens":7,"output_tokens":1}}}`,
			`{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`,
			`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Why "}}`,
			`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"so?"}}`,
			`{"type":"content_block_stop","index":0}`,
			`{"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":5}}`,
			`{"type":"message_stop"}`,
		} {
			fmt.Fprintf(w, "event: x\ndata: %s\n\n", e)
		}
	}))
	defer srv.Close()
	a := NewAnthropic("k", "test-model", srv.URL)
	a.Retry, a.Meter = fast, &Meter{}
	ch, err := a.Stream(context.Background(), domain.LLMRequest{System: "s", Messages: []domain.Message{{Role: "user", Content: "hi"}}})
	if err != nil {
		t.Fatal(err)
	}
	if got := collect(t, ch); got != "Why so?" {
		t.Fatalf("got %q", got)
	}
	if c := a.Meter.Calls(); len(c) != 1 || c[0].In != 7 || c[0].Out != 5 {
		t.Fatalf("meter %+v", c)
	}
}

// Gemini sends an explicit thinking config and the 200 token cap, and can do JSON.
func TestGeminiJSONAndConfig(t *testing.T) {
	var body string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		body = string(b)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"candidates":[{"content":{"role":"model","parts":[{"text":"{\"ok\":true}"}]}}],"usageMetadata":{"promptTokenCount":10,"candidatesTokenCount":3,"thoughtsTokenCount":2}}`)
	}))
	defer srv.Close()
	g, err := NewGemini(context.Background(), "k", "test-model", srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	g.Retry, g.Meter = fast, &Meter{}
	var out struct {
		OK bool `json:"ok"`
	}
	if err := g.JSON(context.Background(), domain.LLMRequest{System: "s", Messages: []domain.Message{{Role: "user", Content: "hi"}}}, &out); err != nil || !out.OK {
		t.Fatalf("err=%v out=%+v", err, out)
	}
	for _, want := range []string{`"thinkingBudget":0`, `"responseJsonSchema"`, `"application/json"`} {
		if !strings.Contains(body, want) {
			t.Errorf("request lacks %s: %s", want, body)
		}
	}
	if c := g.Meter.Calls(); len(c) != 1 || c[0].In != 10 || c[0].Out != 5 {
		t.Fatalf("meter %+v", c)
	}
	if g.MaxTokens != 200 {
		t.Fatalf("stream cap %d", g.MaxTokens)
	}
}

func TestSchemaNullablePointers(t *testing.T) {
	var out struct {
		Results []domain.ClaimResult `json:"results"`
	}
	b, _ := json.Marshal(schemaFor(&out))
	s := string(b)
	if !strings.Contains(s, `"null"`) || strings.Contains(s, "$schema") || strings.Contains(s, "$ref") {
		t.Fatalf("schema: %s", s)
	}
	if schemaFor(map[string]any{}) != nil {
		t.Fatal("non-struct must give no schema")
	}
}

func TestMeterCostAndCap(t *testing.T) {
	m := &Meter{Cap: 0.01}
	if err := m.check(); err != nil {
		t.Fatal(err)
	}
	m.Record(Call{Model: "claude-sonnet-5-5", In: 1_000_000, Out: 1_000_000}) // $2 + $10
	_, _, cost := m.Total()
	if cost != 12 {
		t.Fatalf("cost %v", cost)
	}
	if err := m.check(); !errors.Is(err, ErrBudget) {
		t.Fatalf("want ErrBudget, got %v", err)
	}
	m.Record(Call{Model: "unlisted", In: 5})
	if !strings.Contains(m.Summary(), "no price for unlisted") {
		t.Fatal(m.Summary())
	}
	var nilm *Meter
	nilm.Record(Call{})
	if nilm.check() != nil {
		t.Fatal("nil meter must be a no-op")
	}
}

func TestNewRoutesAndFake(t *testing.T) {
	ctx := context.Background()
	if _, err := New(ctx, "mystery-1", nil); err == nil {
		t.Fatal("unknown model must error")
	}
	t.Setenv("GEMINI_API_KEY", "")
	if _, err := New(ctx, "gemini-x", nil); err == nil || !strings.Contains(err.Error(), "GEMINI_API_KEY") {
		t.Fatalf("err %v", err)
	}
	m := &Meter{}
	f, err := New(ctx, "fake", m)
	if err != nil {
		t.Fatal(err)
	}
	var out struct{ Results []int }
	if err := f.JSON(ctx, domain.LLMRequest{}, &out); err != nil || len(m.Calls()) != 1 {
		t.Fatalf("err=%v calls=%v", err, m.Calls())
	}
	d, err := New(ctx, "fake-down", m)
	if err != nil {
		t.Fatal(err)
	}
	if err := d.JSON(ctx, domain.LLMRequest{}, &out); err == nil {
		t.Fatal("fake-down JSON must fail")
	}
	if _, err := d.Stream(ctx, domain.LLMRequest{}); err == nil {
		t.Fatal("fake-down Stream must fail")
	}
}
