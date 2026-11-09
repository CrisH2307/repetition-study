package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/CrisH2307/repetition-study/internal/checker"
	"github.com/CrisH2307/repetition-study/internal/domain"
)

func TestWorst(t *testing.T) {
	rs := func(ls ...domain.ClaimLabel) (out []domain.ClaimResult) {
		for _, l := range ls {
			out = append(out, domain.ClaimResult{Label: l})
		}
		return
	}
	cases := []struct {
		in   []domain.ClaimResult
		want domain.ClaimLabel
	}{
		{nil, domain.LabelSupported},
		{rs("supported", "partial"), domain.LabelPartial},
		{rs("partial", "absent"), domain.LabelAbsent},
		{rs("absent", "contradicted", "supported"), domain.LabelContradicted},
	}
	for _, c := range cases {
		if got := worst(c.in); got != c.want {
			t.Errorf("worst(%v)=%s want %s", c.in, got, c.want)
		}
	}
}

func TestReport(t *testing.T) {
	s := stats{Model: "m", Proposed: 4, Valid: 3, PriceIn: 1, PriceOut: 2, Calls: []callStat{
		{time.Second, usage{1000, 500}}, {3 * time.Second, usage{3000, 500}},
	}}
	add := func(g, p domain.ClaimLabel, n int) {
		for i := 0; i < n; i++ {
			s.Rows = append(s.Rows, row{"x", "c", g, p})
		}
	}
	add("contradicted", "contradicted", 3)
	add("absent", "partial", 1)
	add("contradicted", "supported", 1)
	add("supported", "supported", 8)
	add("supported", "contradicted", 2)
	out := s.report(time.Unix(0, 0))
	for _, want := range []string{
		"flagged contradicted or absent: 3 of 5",
		"falsely flagged contradicted: 2 of 10",
		"75.0% (3 of 4)",
		"| contradicted | 1 | 0 | 3 | 0 |",
		"| supported | 8 | 0 | 2 | 0 |",
		"mean 2s, median 3s, max 3s",
		"mean 2000 in, 500 out",
		"$0.00300 per call, $0.0060 total",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("report missing %q\n%s", want, out)
		}
	}
	if strings.Contains(strings.ToLower(out), "winner") {
		t.Error("report must not pick a winner")
	}
}

func TestLoadLabelsRejectsIncomplete(t *testing.T) {
	dir := t.TempDir()
	good := filepath.Join(dir, "g.csv")
	bad := filepath.Join(dir, "b.csv")
	os.WriteFile(good, []byte("id,label\ne1,supported\ne2, Contradicted\n"), 0o644)
	os.WriteFile(bad, []byte("id,label\ne1,\n"), 0o644)
	m, err := loadLabels(good)
	if err != nil || m["e2"] != domain.LabelContradicted {
		t.Fatalf("%v %v", m, err)
	}
	if _, err := loadLabels(bad); err == nil {
		t.Fatal("empty label must error")
	}
}

func TestDecodeFenced(t *testing.T) {
	var out struct{ A int }
	if err := decode("```json\n{\"A\": 2}\n```", &out); err != nil || out.A != 2 {
		t.Fatalf("%v %v", out, err)
	}
	if decode("no json", &out) == nil {
		t.Fatal("want error")
	}
}

type cannedProvider struct{ reply string }

func (c cannedProvider) json(_ context.Context, _ domain.LLMRequest, out any) (usage, error) {
	return usage{10, 5}, decode(c.reply, out)
}

// The meter records one call per explanation and the checker feeds the stats.
func TestMeterAndCheckerWiring(t *testing.T) {
	m := &meter{p: cannedProvider{`{"results":[{"claim_id":"k1","student_text":"s","label":"contradicted","correction":"c","citation":{"source_id":"s1","quote":"not in source","locator":""}}]}`}}
	ck := &checker.Checker{LLM: m, AllowAbsent: true}
	o, err := ck.CheckDetailed(context.Background(), domain.CheckInput{
		Concept: domain.Concept{ID: "c", Claims: []domain.Claim{{ID: "k1", Required: true}, {ID: "k2", Required: true}}},
		Sources: []domain.Source{{ID: "s1", Kind: domain.SourceCourse, Text: "real text"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(m.calls) != 1 || m.calls[0].Usage.In != 10 {
		t.Fatalf("calls %+v", m.calls)
	}
	// Bad quote: contradicted falls to partial, k2 is filled as absent.
	if worst(o.Results) != domain.LabelAbsent || o.Proposed != 1 || o.Valid != 0 {
		t.Fatalf("%+v", o)
	}
}
