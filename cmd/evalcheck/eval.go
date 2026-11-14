package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/CrisH2307/repetition-study/internal/domain"
)

type unit struct {
	Sources  []domain.Source  `json:"sources"`
	Concepts []domain.Concept `json:"concepts"`
}

type explanation struct {
	ID        string `json:"id"`
	ConceptID string `json:"concept_id"`
	Text      string `json:"text"`
}

type perturbation struct {
	ID        string `json:"id"`
	ConceptID string `json:"concept_id"`
	ClaimID   string `json:"claim_id"`
	ErrorType string `json:"error_type"`
	Rule      string `json:"rule"`
	Text      string `json:"text"`
}

type injection struct {
	ID            string `json:"id"`
	ExplanationID string `json:"explanation_id"`
	Kind          string `json:"kind"`
	Place         string `json:"place"` // append, prepend or latex
	Attack        string `json:"attack"`
}

type coverageRow struct {
	ID        string `json:"id"`
	ConceptID string `json:"concept_id"`
	Coverage  int    `json:"coverage"` // percent of the concept's claims the text states
	Text      string `json:"text"`
}

type leakSession struct {
	ID        string `json:"id"`
	ConceptID string `json:"concept_id"`
	Teach     string `json:"teach"`
	Probe     string `json:"probe"`
}

type fileHash struct {
	Path, SHA string
	Rows      int
}

// labelKey is one row of labels_final.csv: explanation id, claim id.
type labelKey struct{ Expl, Claim string }

// dataset is everything a run reads. A set that is missing has an empty slice
// and a *Why saying so; its suites are skipped with that reason, never passed.
type dataset struct {
	Unit     unit
	Concepts map[string]domain.Concept
	Split    string

	Expl    []explanation          // the split's explanations
	AllExpl map[string]explanation // every explanation, for injection bases
	ExplWhy string

	Labels    map[labelKey]domain.ClaimLabel
	LabelsWhy string

	Perturb    []perturbation
	PerturbWhy string
	Inject     []injection
	InjectWhy  string
	Coverage   []coverageRow
	CoverWhy   string
	Leak       []leakSession
	LeakWhy    string

	Files []fileHash
}

// read returns nil bytes and a reason when the file does not exist.
func (d *dataset) read(path string) ([]byte, string, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, path + " does not exist", nil
	}
	if err != nil {
		return nil, "", err
	}
	d.Files = append(d.Files, fileHash{Path: path, SHA: fmt.Sprintf("%x", sha256.Sum256(b)), Rows: bytes.Count(b, []byte("\n"))})
	return b, "", nil
}

func jsonl[T any](path string, b []byte) ([]T, error) {
	var out []T
	sc := bufio.NewScanner(bytes.NewReader(b))
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for n := 1; sc.Scan(); n++ {
		if strings.TrimSpace(sc.Text()) == "" {
			continue
		}
		var v T
		if err := json.Unmarshal(sc.Bytes(), &v); err != nil {
			return nil, fmt.Errorf("%s line %d: %w", path, n, err)
		}
		out = append(out, v)
	}
	return out, sc.Err()
}

// loadSet reads one optional JSONL set.
func loadSet[T any](d *dataset, path string) ([]T, string, error) {
	b, why, err := d.read(path)
	if err != nil || why != "" {
		return nil, why, err
	}
	rows, err := jsonl[T](path, b)
	if err == nil && len(rows) == 0 {
		why = path + " is empty"
	}
	return rows, why, err
}

// loadData reads the unit and every set under dir. split is test, dev or all;
// it filters explanations and perturbations through split.json.
func loadData(unitPath, dir, labelsPath, split string) (*dataset, error) {
	d := &dataset{Concepts: map[string]domain.Concept{}, AllExpl: map[string]explanation{}, Split: split}
	b, why, err := d.read(unitPath)
	if err != nil {
		return nil, err
	}
	if why != "" {
		return nil, errors.New(why)
	}
	if err := json.Unmarshal(b, &d.Unit); err != nil {
		return nil, fmt.Errorf("%s: %w", unitPath, err)
	}
	claims := map[string]map[string]bool{}
	for _, c := range d.Unit.Concepts {
		d.Concepts[c.ID] = c
		claims[c.ID] = map[string]bool{}
		for _, k := range c.Claims {
			claims[c.ID][k.ID] = true
		}
	}
	known := func(file, id, concept string) error {
		if _, ok := d.Concepts[concept]; !ok {
			return fmt.Errorf("%s: %s names concept %q, which is not in %s", file, id, concept, unitPath)
		}
		return nil
	}

	keep := map[string]map[string]bool{} // set name -> ids in the split
	if split != "all" {
		sb, why, err := d.read(filepath.Join(dir, "split.json"))
		if err != nil {
			return nil, err
		}
		if why != "" {
			return nil, fmt.Errorf("%s (needed for --split %s; use --split all to run without it)", why, split)
		}
		var sp map[string]json.RawMessage
		if err := json.Unmarshal(sb, &sp); err != nil {
			return nil, fmt.Errorf("split.json: %w", err)
		}
		for _, set := range []string{"explanations", "perturbations"} {
			var parts map[string][]string
			if err := json.Unmarshal(sp[set], &parts); err != nil {
				return nil, fmt.Errorf("split.json %s: %w", set, err)
			}
			ids, ok := parts[split]
			if !ok {
				return nil, fmt.Errorf("split.json has no %q part for %s (use test, dev or all)", split, set)
			}
			keep[set] = map[string]bool{}
			for _, id := range ids {
				keep[set][id] = true
			}
		}
	}
	in := func(set, id string) bool { return split == "all" || keep[set][id] }

	p := filepath.Join(dir, "explanations.jsonl")
	all, why, err := loadSet[explanation](d, p)
	if err != nil {
		return nil, err
	}
	d.ExplWhy = why
	for _, e := range all {
		if err := known(p, e.ID, e.ConceptID); err != nil {
			return nil, err
		}
		d.AllExpl[e.ID] = e
		if in("explanations", e.ID) {
			d.Expl = append(d.Expl, e)
		}
	}
	if why == "" && len(d.Expl) == 0 {
		d.ExplWhy = fmt.Sprintf("no explanation is in the %s split", split)
	}

	if d.ExplWhy != "" {
		d.LabelsWhy = d.ExplWhy
	} else if d.Labels, d.LabelsWhy, err = d.loadLabels(labelsPath); err != nil {
		return nil, err
	}

	p = filepath.Join(dir, "perturbations.jsonl")
	ps, why, err := loadSet[perturbation](d, p)
	if err != nil {
		return nil, err
	}
	d.PerturbWhy = why
	for _, x := range ps {
		if err := known(p, x.ID, x.ConceptID); err != nil {
			return nil, err
		}
		if !claims[x.ConceptID][x.ClaimID] {
			return nil, fmt.Errorf("%s: %s names claim %q, which is not in its concept", p, x.ID, x.ClaimID)
		}
		if in("perturbations", x.ID) {
			d.Perturb = append(d.Perturb, x)
		}
	}
	if why == "" && len(d.Perturb) == 0 {
		d.PerturbWhy = fmt.Sprintf("no perturbation is in the %s split", split)
	}

	p = filepath.Join(dir, "injections.jsonl")
	if d.Inject, d.InjectWhy, err = loadSet[injection](d, p); err != nil {
		return nil, err
	}
	for _, x := range d.Inject {
		if _, ok := d.AllExpl[x.ExplanationID]; !ok {
			return nil, fmt.Errorf("%s: %s rides on explanation %q, which is not in explanations.jsonl", p, x.ID, x.ExplanationID)
		}
		if x.Place != "append" && x.Place != "prepend" && x.Place != "latex" {
			return nil, fmt.Errorf("%s: %s has place %q, want append, prepend or latex", p, x.ID, x.Place)
		}
	}

	p = filepath.Join(dir, "coverage.jsonl")
	if d.Coverage, d.CoverWhy, err = loadSet[coverageRow](d, p); err != nil {
		return nil, err
	}
	for _, x := range d.Coverage {
		if err := known(p, x.ID, x.ConceptID); err != nil {
			return nil, err
		}
	}

	p = filepath.Join(dir, "leak_sessions.jsonl")
	if d.Leak, d.LeakWhy, err = loadSet[leakSession](d, p); err != nil {
		return nil, err
	}
	for _, x := range d.Leak {
		if err := known(p, x.ID, x.ConceptID); err != nil {
			return nil, err
		}
	}
	return d, nil
}

// loadLabels reads the frozen claim labels (columns id, claim_id, label). The
// file counts only if FROZEN.md next to it records its hash, so an edited or
// unfrozen file skips the human suite instead of scoring against it.
func (d *dataset) loadLabels(path string) (map[labelKey]domain.ClaimLabel, string, error) {
	b, why, err := d.read(path)
	if err != nil {
		return nil, "", err
	}
	if why != "" {
		return nil, why + " (the labels are not frozen yet)", nil
	}
	sha := d.Files[len(d.Files)-1].SHA
	frozen := filepath.Join(filepath.Dir(path), "FROZEN.md")
	if fb, err := os.ReadFile(frozen); err != nil || !bytes.Contains(fb, []byte(sha)) {
		return nil, fmt.Sprintf("%s is not the frozen set: %s does not record its SHA-256 %s", path, frozen, sha), nil
	}
	rows, err := csv.NewReader(bytes.NewReader(b)).ReadAll()
	if err != nil {
		return nil, "", fmt.Errorf("%s: %w", path, err)
	}
	if len(rows) == 0 {
		return nil, "", fmt.Errorf("%s is empty", path)
	}
	col := map[string]int{"id": -1, "claim_id": -1, "label": -1}
	for i, h := range rows[0] {
		if _, ok := col[strings.ToLower(strings.TrimSpace(h))]; ok {
			col[strings.ToLower(strings.TrimSpace(h))] = i
		}
	}
	for name, i := range col {
		if i < 0 {
			return nil, "", fmt.Errorf("%s needs header columns id, claim_id and label (no %s)", path, name)
		}
	}
	out := map[labelKey]domain.ClaimLabel{}
	for n, r := range rows[1:] {
		if len(r) <= col["id"] || len(r) <= col["claim_id"] || len(r) <= col["label"] {
			return nil, "", fmt.Errorf("%s row %d has %d columns, need id, claim_id and label", path, n+2, len(r))
		}
		k := labelKey{strings.TrimSpace(r[col["id"]]), strings.TrimSpace(r[col["claim_id"]])}
		l := domain.ClaimLabel(strings.ToLower(strings.TrimSpace(r[col["label"]])))
		switch l {
		case domain.LabelSupported, domain.LabelPartial, domain.LabelContradicted, domain.LabelAbsent:
			out[k] = l
		default:
			return nil, "", fmt.Errorf("%s: %s claim %s has label %q, labels must be complete", path, k.Expl, k.Claim, r[col["label"]])
		}
	}
	for _, e := range d.Expl {
		for _, c := range d.Concepts[e.ConceptID].Claims {
			if _, ok := out[labelKey{e.ID, c.ID}]; !ok {
				return nil, "", fmt.Errorf("%s has no label for explanation %s, claim %s", path, e.ID, c.ID)
			}
		}
	}
	return out, "", nil
}

// worst collapses results into one label.
// Order: contradicted > absent > partial > supported. Contradicted and absent
// both map to FSRS Again, so ranking them apart only breaks ties.
func worst(rs []domain.ClaimResult) domain.ClaimLabel {
	rank := map[domain.ClaimLabel]int{domain.LabelSupported: 0, domain.LabelPartial: 1, domain.LabelAbsent: 2, domain.LabelContradicted: 3}
	w := domain.LabelSupported
	for _, r := range rs {
		if rank[r.Label] > rank[w] {
			w = r.Label
		}
	}
	return w
}

// labelsOf gives the checker's label for every claim of a concept: the worst
// result that names the claim, or absent when no result names it.
func labelsOf(c domain.Concept, rs []domain.ClaimResult) map[string]domain.ClaimLabel {
	by := map[string][]domain.ClaimResult{}
	for _, r := range rs {
		if r.ClaimID != nil {
			by[*r.ClaimID] = append(by[*r.ClaimID], r)
		}
	}
	out := map[string]domain.ClaimLabel{}
	for _, k := range c.Claims {
		out[k.ID] = domain.LabelAbsent
		if g := by[k.ID]; len(g) > 0 {
			out[k.ID] = worst(g)
		}
	}
	return out
}

// resolveUnit picks the real unit from gitignored content/private/ unless
// --unit is given. content/unit.json has no source text, so it is not a default.
func resolveUnit(flagPath string) (string, error) {
	if flagPath != "" {
		return flagPath, nil
	}
	const def = "content/private/unit.json"
	if _, err := os.Stat(def); err != nil {
		return "", fmt.Errorf("%s not found: the real course unit is never committed. See content/README.md, or pass --unit content/unit.json for a fake-model dry run", def)
	}
	return def, nil
}
