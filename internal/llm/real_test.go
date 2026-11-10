//go:build realllm

package llm

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/CrisH2307/repetition-study/internal/domain"
)

// Real-API checks. They spend money, so they sit behind the realllm tag and a
// US$5 cap for this unit. Run only with Cris's approval:
//
//	GEMINI_API_KEY=... STUDENT_MODEL=... ANTHROPIC_API_KEY=... CHECKER_MODEL=... \
//	  go test -tags realllm ./internal/llm/ -run Real -v
//
// Each vendor is tested in both roles, because either may fill either.
func realClients(t *testing.T, meter *Meter) map[string]domain.LLM {
	t.Helper()
	ctx := context.Background()
	out := map[string]domain.LLM{}
	if k, m := os.Getenv("GEMINI_API_KEY"), os.Getenv("STUDENT_MODEL"); k != "" && m != "" {
		g, err := NewGemini(ctx, k, m, "")
		if err != nil {
			t.Fatal(err)
		}
		g.Meter = meter
		out["gemini"] = g
	}
	if k, m := os.Getenv("ANTHROPIC_API_KEY"), os.Getenv("CHECKER_MODEL"); k != "" && m != "" {
		a := NewAnthropic(k, m, "")
		a.Meter = meter
		out["anthropic"] = a
	}
	if len(out) == 0 {
		t.Skip("no API keys and model ids in the environment")
	}
	return out
}

func TestRealStreamAndJSON(t *testing.T) {
	meter := &Meter{Cap: 5}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	for name, c := range realClients(t, meter) {
		t.Run(name+"/stream", func(t *testing.T) {
			ch, err := c.Stream(ctx, domain.LLMRequest{System: "You are a curious student. Ask one short question.", Messages: []domain.Message{{Role: "user", Content: "A derivative is a slope."}}})
			if err != nil {
				t.Fatal(err)
			}
			var b strings.Builder
			for d := range ch {
				if d.Err != nil {
					t.Fatal(d.Err)
				}
				b.WriteString(d.Text)
			}
			if strings.TrimSpace(b.String()) == "" {
				t.Fatal("empty reply: with thinking on, the cap can be eaten before any text (F10)")
			}
		})
		t.Run(name+"/json", func(t *testing.T) {
			var out struct {
				Answer string  `json:"answer"`
				Note   *string `json:"note"`
			}
			err := c.JSON(ctx, domain.LLMRequest{System: "Answer with the fields answer and note (null if none).", Messages: []domain.Message{{Role: "user", Content: "What is 2+2?"}}}, &out)
			if err != nil || out.Answer == "" {
				t.Fatalf("err=%v out=%+v", err, out)
			}
		})
	}
	t.Log("cost:", meter.Summary())
}
