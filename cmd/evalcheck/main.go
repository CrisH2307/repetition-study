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
	"github.com/CrisH2307/repetition-study/internal/llm"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "evalcheck:", err)
		os.Exit(1)
	}
}

func run() error {
	model := flag.String("model", "", "model name: gemini-*, claude-* or fake (required)")
	unitPath := flag.String("unit", "", "course unit with sources and concepts (default content/private/unit.json)")
	expPath := flag.String("explanations", "eval/data/explanations.jsonl", "explanations")
	lblPath := flag.String("labels", "eval/data/labels_final.csv", "final human labels")
	outDir := flag.String("out", "eval/results", "results directory")
	maxCost := flag.Float64("max-cost", 5, "stop the run when metered cost reaches this many USD (0 = no cap)")
	flag.Parse()
	if *model == "" {
		return fmt.Errorf("--model is required")
	}

	up, err := resolveUnit(*unitPath)
	if err != nil {
		return err
	}
	u, err := loadUnit(up)
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
	// Same constructor the server uses, so a pass here certifies the shipped path.
	meter := &llm.Meter{Cap: *maxCost}
	l, err := llm.New(ctx, *model, meter)
	if err != nil {
		return err
	}
	// Each explanation is a finished session, so absent is allowed.
	ck := &checker.Checker{LLM: l, Model: *model, AllowAbsent: true}
	st := stats{Model: *model}
	for _, e := range exps {
		o, err := ck.CheckDetailed(ctx, domain.CheckInput{Concept: concepts[e.ConceptID], Sources: u.Sources, Transcript: e.Text})
		if err != nil {
			return fmt.Errorf("explanation %q: %w", e.ID, err)
		}
		st.Proposed += o.Proposed
		st.Valid += o.Valid
		st.Rows = append(st.Rows, row{e.ID, e.ConceptID, labels[e.ID], worst(o.Results)})
	}
	st.Calls = meter.Calls()
	st.Models = meter.Models

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		return err
	}
	name := strings.NewReplacer("/", "_", " ", "_").Replace(*model) + ".md"
	path := filepath.Join(*outDir, name)
	if err := os.WriteFile(path, []byte(st.report(time.Now())), 0o644); err != nil {
		return err
	}
	fmt.Println("wrote", path)
	fmt.Println("cost per session:", meter.Summary())
	return nil
}

// resolveUnit picks the real unit from gitignored content/private/ unless
// --unit is given. content/unit.json has no source text, so it is not a default.
func resolveUnit(flagPath string) (string, error) {
	if flagPath != "" {
		return flagPath, nil
	}
	const def = "content/private/unit.json"
	if _, err := os.Stat(def); err != nil {
		return "", fmt.Errorf("%s not found: the real course unit is never committed. See content/README.md, or pass --unit content/fixture/unit.json to try the fixture", def)
	}
	return def, nil
}
