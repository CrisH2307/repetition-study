// Command evalcheck runs the gate G0 suites (docs/PLAN.md) and writes gate.md.
// It reports numbers only and never picks a model.
package main

import (
	"context"
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/CrisH2307/repetition-study/internal/checker"
	"github.com/CrisH2307/repetition-study/internal/llm"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "evalcheck:", err)
		os.Exit(1)
	}
}

func list(s string) []string {
	var out []string
	for _, x := range strings.Split(s, ",") {
		if x = strings.TrimSpace(x); x != "" {
			out = append(out, x)
		}
	}
	return out
}

func run(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("evalcheck", flag.ContinueOnError)
	var models string
	fs.StringVar(&models, "models", "", "checker models to test, comma separated: gemini-*, claude-* or fake")
	fs.StringVar(&models, "model", "", "same as --models")
	student := fs.String("student-model", os.Getenv("STUDENT_MODEL"), "AI student model for the leak suite (default $STUDENT_MODEL)")
	suites := fs.String("suites", strings.Join(suiteOrder, ","), "suites to run, comma separated")
	unitPath := fs.String("unit", "", "course unit with sources and concepts (default content/private/unit.json)")
	dataDir := fs.String("data", "eval/data", "directory with the eval sets")
	lblPath := fs.String("labels", "", "frozen human labels (default <data>/labels_final.csv)")
	split := fs.String("split", "test", "which part of split.json to score: test, dev or all")
	outDir := fs.String("out", "eval/results", "where gate.md is written")
	privDir := fs.String("private-out", "content/private/eval", "where details that can hold course text are written (never committed)")
	maxCost := fs.Float64("max-cost", 5, "stop before metered cost passes this many USD, all models and suites together (0 = no cap)")
	turns := fs.Int("turns", 8, "AI student turns counted in one session for the cost row")
	if err := fs.Parse(args); err != nil {
		return err
	}
	want := list(*suites)
	for _, s := range want {
		if !slices.Contains(suiteOrder, s) {
			return fmt.Errorf("unknown suite %q, want any of %s", s, strings.Join(suiteOrder, ","))
		}
	}
	on := func(s string) bool { return slices.Contains(want, s) }
	ms := list(models)
	if len(ms) == 0 && !(on("leak") && *student != "") {
		return errors.New("nothing to run: give --models, or --student-model with the leak suite")
	}
	offline := func(m string) bool { return m == "" || m == "fake" || m == "fake-down" }
	all := slices.Concat(ms, []string{*student})
	for _, m := range all {
		if *maxCost > 0 && !offline(m) && !llm.DefaultModels().Priced(m) {
			return fmt.Errorf("%s has no price in internal/llm/models.json, so the spend cap cannot hold. Add a verified price, or pass --max-cost 0 to run with no cap", m)
		}
	}
	if *lblPath == "" {
		*lblPath = filepath.Join(*dataDir, "labels_final.csv")
	}

	up, err := resolveUnit(*unitPath)
	if err != nil {
		return err
	}
	d, err := loadData(up, *dataDir, *lblPath, *split)
	if err != nil {
		return err
	}
	real := slices.ContainsFunc(all, func(m string) bool { return !offline(m) })
	for _, s := range d.Unit.Sources {
		if real && strings.TrimSpace(s.Text) == "" {
			return fmt.Errorf("source %q in %s has no text: a real model needs the real unit (content/private/unit.json)", s.ID, up)
		}
	}

	ctx := context.Background()
	// One meter for the whole run, so the cap covers every model and suite.
	meter := &llm.Meter{Cap: *maxCost, Models: llm.DefaultModels()}
	bud := &budget{meter: meter, cap: *maxCost}
	rep := report{When: time.Now(), Commit: commit(), Command: "evalcheck " + strings.Join(args, " "), Split: *split,
		StudentModel: *student, Cap: *maxCost, Turns: *turns, PrivateDir: *privDir}
	for _, s := range suiteOrder {
		if on(s) {
			rep.Suites = append(rep.Suites, s)
		}
	}

	// Leak first: the cost row needs the student's cost per turn.
	var lk leakRun
	if on("leak") {
		switch {
		case *student == "":
			lk.res = skip("leak", "no student model was given (--student-model or STUDENT_MODEL)")
		default:
			// Same constructor the server uses, so a pass here certifies the shipped path.
			sl, err := llm.New(ctx, *student, meter)
			if err != nil {
				return err
			}
			fmt.Fprintf(stdout, "leak suite, student model %s...\n", *student)
			lk = leak(ctx, newGuard(sl, *student, bud), d, bud)
		}
		rep.Leak = &lk.res
	}

	var sheet [][]string
	for _, m := range ms {
		l, err := llm.New(ctx, m, meter)
		if err != nil {
			return err
		}
		r := &runner{ctx: ctx, model: m, d: d, bud: bud,
			ck: &checker.Checker{LLM: newGuard(l, m, bud), Model: m, AllowAbsent: true, Log: quiet}}
		mr := modelReport{Model: m}
		steps := []struct {
			key string
			fn  func() result
		}{
			{"human", r.human}, {"perturb", r.perturb}, {"citations", r.citations}, {"stability", r.stability},
			{"injection", r.injection}, {"coverage", r.coverage},
			{"cost", func() result { return r.cost(meter.Models, lk, *student, *turns) }},
		}
		for _, s := range steps {
			if !on(s.key) {
				continue
			}
			fmt.Fprintf(stdout, "%s, %s...\n", m, s.key)
			mr.Results = append(mr.Results, s.fn())
		}
		rep.Models = append(rep.Models, mr)
		sheet = append(sheet, r.sheet...)
	}
	if len(rep.Models) == 0 {
		rep.Models = []modelReport{{Model: "no checker model"}}
	}
	_, _, rep.Spent = meter.Total()
	rep.CapHit = bud.hit
	rep.Files = d.Files

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(*privDir, 0o755); err != nil {
		return err
	}
	gate := filepath.Join(*outDir, "gate.md")
	if err := os.WriteFile(gate, []byte(rep.markdown()), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(*privDir, "gate_private.md"), []byte(rep.private()), 0o644); err != nil {
		return err
	}
	if on("citations") && len(ms) > 0 {
		if err := writeSheet(filepath.Join(*privDir, "citations_to_rate.csv"), sheet); err != nil {
			return err
		}
	}
	fmt.Fprintf(stdout, "\n%s\nwrote %s\nprivate details in %s\nspend: %s\n", rep.summary(), gate, *privDir, meter.Summary())
	if bud.hit {
		return fmt.Errorf("stopped at the spend cap of $%.2f; %s has what was measured", *maxCost, gate)
	}
	return nil
}

// writeSheet writes every shipped citation for a person to mark yes or no.
func writeSheet(path string, rows [][]string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	w.Write([]string{"model", "explanation", "claim_id", "claim", "student_text", "label", "correction", "quote", "locator", "relevant"})
	w.WriteAll(rows)
	return w.Error()
}

// commit is the checked-out commit, marked when the tree has local changes.
func commit() string {
	out, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil {
		return "unknown (git not available)"
	}
	c := strings.TrimSpace(string(out))
	if st, err := exec.Command("git", "status", "--porcelain").Output(); err == nil && len(st) > 0 {
		c += " plus uncommitted changes"
	}
	return c
}
