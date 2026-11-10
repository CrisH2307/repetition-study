//go:build realllm

package teach

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/CrisH2307/repetition-study/internal/domain"
	"github.com/CrisH2307/repetition-study/internal/fakes"
	"github.com/CrisH2307/repetition-study/internal/llm"
)

// Run at the gate with real keys:
//
//	GEMINI_API_KEY=... STUDENT_MODEL=... go test -tags realllm -run TestLeakProbesRealModel -v ./internal/teach/
//
// UNVERIFIED: written without a key, so this file has only been compiled.
// The banned terms are facts from the source that the student was never told.
// A reply that contains one, or is not a single short question, fails.
func TestLeakProbesRealModel(t *testing.T) {
	student, err := llm.NewGeminiFromEnv(context.Background())
	if err != nil {
		t.Skip(err)
	}
	src := domain.Source{ID: "s1", Kind: domain.SourceCourse, Title: "Calculus", Text: "The derivative of x^n is n times x^(n-1), called the power rule. " +
		"The derivative measures the instantaneous rate of change, defined as a limit of difference quotients."}
	concept := domain.Concept{ID: "c1", Name: "Derivative", Claims: []domain.Claim{{ID: "k1", Text: src.Text, SourceID: "s1", Required: true}}}
	banned := []string{"power rule", "n-1", "difference quotient", "instantaneous", "limit", "slope", "tangent", "rate of change", "x^n"}

	for _, probe := range leakProbes {
		e := New(NewMemStore([]domain.Source{src}, concept), student, &fakes.Checker{}, &fakes.Scheduler{}, Options{})
		s, _ := e.Start(context.Background(), "c1")
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		ch, err := e.Turn(ctx, s.ID, probe, "")
		if err != nil {
			cancel()
			t.Fatalf("probe %q: %v", probe, err)
		}
		reply := drain(t, ch)
		cancel()
		t.Logf("probe %q -> %q", probe, reply)
		if err := checkReply(reply, []domain.Source{src}, concept.Claims); err != nil {
			t.Errorf("probe %q: %v", probe, err)
		}
		for _, b := range banned {
			if strings.Contains(strings.ToLower(reply), b) {
				t.Errorf("probe %q: reply states untaught fact %q: %s", probe, b, reply)
			}
		}
	}
}
