//go:build realllm

package app

import (
	"context"
	"os"
	"testing"

	"github.com/CrisH2307/repetition-study/internal/llm"
)

// TestFullLoopRealModels is the fake-model loop with the real student and
// checker. It spends money, so it needs Cris's approval and both keys:
//
//	GEMINI_API_KEY=... STUDENT_MODEL=... ANTHROPIC_API_KEY=... CHECKER_MODEL=... \
//	  go test -tags realllm ./internal/app/ -run RealModels -v
//
// UNVERIFIED: written without keys, so it has only been compiled. The model
// decides the labels, so a failure may be the model's output, not a code bug.
func TestFullLoopRealModels(t *testing.T) {
	sm, cm := os.Getenv("STUDENT_MODEL"), os.Getenv("CHECKER_MODEL")
	if sm == "" || cm == "" {
		t.Skip("STUDENT_MODEL and CHECKER_MODEL are not set")
	}
	meter := &llm.Meter{Cap: 1} // US$1 ceiling for the whole test
	student, err := llm.New(context.Background(), sm, meter)
	if err != nil {
		t.Skip(err)
	}
	checkerLLM, err := llm.New(context.Background(), cm, meter)
	if err != nil {
		t.Skip(err)
	}
	runLoop(t, newEnv(t, student, checkerLLM, cm))
	t.Log(meter.Summary())
}
