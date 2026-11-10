// Command evalcheck runs the checker over the labelled explanations and writes
// eval/results/<model>.md. It reports numbers only and never picks a model.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/CrisH2307/repetition-study/internal/checker"
	"github.com/CrisH2307/repetition-study/internal/domain"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "evalcheck:", err)
		os.Exit(1)
	}
}

func run() error {
	model := flag.String("model", "", "model name: gemini-*, claude-* or fake (required)")
	unitPath := flag.String("unit", "content/unit.json", "course unit with sources and concepts")
	expPath := flag.String("explanations", "eval/data/explanations.jsonl", "explanations")
	lblPath := flag.String("labels", "eval/data/labels_final.csv", "final human labels")
	outDir := flag.String("out", "eval/results", "results directory")
	pin := flag.Float64("price-in", 0, "USD per million input tokens (optional, enables cost)")
	pout := flag.Float64("price-out", 0, "USD per million output tokens (optional, enables cost)")
	flag.Parse()
	if *model == "" {
		return fmt.Errorf("--model is required")
	}

	u, err := loadUnit(*unitPath)
	if err != nil {
		return err
	}
	exps, err := loadExplanations(*expPath)
	if err != nil {
		return err
	}
	labels, err := loadLabels(*lblPath)
	if err != nil {
		return err
	}
	concepts := map[string]domain.Concept{}
	for _, c := range u.Concepts {
		concepts[c.ID] = c
	}
	for _, e := range exps {
		if _, ok := labels[e.ID]; !ok {
			return fmt.Errorf("explanation %q has no row in %s", e.ID, *lblPath)
		}
		if _, ok := concepts[e.ConceptID]; !ok {
			return fmt.Errorf("explanation %q names unknown concept %q", e.ID, e.ConceptID)
		}
	}

	ctx := context.Background()
	p, err := newProvider(ctx, *model)
	if err != nil {
		return err
	}
	m := &meter{p: p}
	// Each explanation is a finished session, so absent is allowed.
	ck := &checker.Checker{LLM: m, Model: *model, AllowAbsent: true}
	st := stats{Model: *model, PriceIn: *pin, PriceOut: *pout}
	for _, e := range exps {
		o, err := ck.CheckDetailed(ctx, domain.CheckInput{Concept: concepts[e.ConceptID], Sources: u.Sources, Transcript: e.Text})
		if err != nil {
			return fmt.Errorf("explanation %q: %w", e.ID, err)
		}
		st.Proposed += o.Proposed
		st.Valid += o.Valid
		st.Rows = append(st.Rows, row{e.ID, e.ConceptID, labels[e.ID], worst(o.Results)})
	}
	st.Calls = m.calls

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		return err
	}
	name := strings.NewReplacer("/", "_", " ", "_").Replace(*model) + ".md"
	path := filepath.Join(*outDir, name)
	if err := os.WriteFile(path, []byte(st.report(time.Now())), 0o644); err != nil {
		return err
	}
	fmt.Println("wrote", path)
	return nil
}
