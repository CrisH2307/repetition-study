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
	"github.com/CrisH2307/repetition-study/internal/llm"
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
	ms, _ := llm.LoadModels([]byte(`{"models":{"m":{"vendor":"gemini","in_per_mtok":1,"out_per_mtok":2}}}`))
	meter := &llm.Meter{Models: ms}
	meter.Record(llm.Call{Model: "m", In: 1000, Out: 500, Latency: time.Second})
	meter.Record(llm.Call{Model: "m", In: 3000, Out: 500, Latency: 3 * time.Second})
	s := stats{Model: "m", Proposed: 4, Valid: 3, Models: ms, Calls: meter.Calls()}
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

func TestLoadLabelsRaggedRow(t *testing.T) {
	p := filepath.Join(t.TempDir(), "r.csv")
	os.WriteFile(p, []byte("note,id,label\nx,e1\n"), 0o644)
	if _, err := loadLabels(p); err == nil {
		t.Fatal("short row must error, not panic")
	}
}

func TestResolveUnit(t *testing.T) {
	t.Chdir(t.TempDir())
	if _, err := resolveUnit(""); err == nil || !strings.Contains(err.Error(), "content/README.md") {
		t.Fatalf("want pointer to content/README.md, got %v", err)
	}
	os.MkdirAll("content/private", 0o755)
	os.WriteFile("content/private/unit.json", []byte("{}"), 0o644)
	if p, err := resolveUnit(""); err != nil || p != "content/private/unit.json" {
		t.Fatalf("%q %v", p, err)
	}
	if p, _ := resolveUnit("x.json"); p != "x.json" {
		t.Fatal("flag must win")
	}
}

// The checker runs through llm.New("fake"): same constructor as the server.
func TestRunFakeThroughInternalLLM(t *testing.T) {
	meter := &llm.Meter{}
	l, err := llm.New(context.Background(), "fake", meter)
	if err != nil {
		t.Fatal(err)
	}
	ck := &checker.Checker{LLM: l, AllowAbsent: true}
	o, err := ck.CheckDetailed(context.Background(), domain.CheckInput{
		Concept: domain.Concept{ID: "c", Claims: []domain.Claim{{ID: "k1", Required: true}}},
		Sources: []domain.Source{{ID: "s1", Kind: domain.SourceCourse, Text: "real text"}},
	})
	if err != nil || len(meter.Calls()) != 1 || worst(o.Results) != domain.LabelAbsent {
		t.Fatalf("err=%v calls=%v o=%+v", err, meter.Calls(), o)
	}
}
