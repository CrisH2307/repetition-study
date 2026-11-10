package main

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/CrisH2307/repetition-study/internal/domain"
	"github.com/CrisH2307/repetition-study/internal/llm"
)

// Input shapes are UNVERIFIED: U1 had not merged eval/data when this was written.
type unit struct {
	Sources  []domain.Source  `json:"sources"`
	Concepts []domain.Concept `json:"concepts"`
}

type explanation struct {
	ID        string `json:"id"`
	ConceptID string `json:"concept_id"`
	Text      string `json:"text"`
}

func loadUnit(path string) (unit, error) {
	var u unit
	b, err := os.ReadFile(path)
	if err != nil {
		return u, err
	}
	return u, json.Unmarshal(b, &u)
}

func loadExplanations(path string) ([]explanation, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []explanation
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for n := 1; sc.Scan(); n++ {
		if strings.TrimSpace(sc.Text()) == "" {
			continue
		}
		var e explanation
		if err := json.Unmarshal(sc.Bytes(), &e); err != nil {
			return nil, fmt.Errorf("%s line %d: %w", path, n, err)
		}
		out = append(out, e)
	}
	return out, sc.Err()
}

// loadLabels reads a CSV with header columns "id" and "label".
func loadLabels(path string) (map[string]domain.ClaimLabel, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("%s is empty", path)
	}
	idCol, lblCol := -1, -1
	for i, h := range rows[0] {
		switch strings.ToLower(strings.TrimSpace(h)) {
		case "id":
			idCol = i
		case "label":
			lblCol = i
		}
	}
	if idCol < 0 || lblCol < 0 {
		return nil, fmt.Errorf("%s needs header columns id and label", path)
	}
	out := map[string]domain.ClaimLabel{}
	for n, r := range rows[1:] {
		if len(r) <= idCol || len(r) <= lblCol {
			return nil, fmt.Errorf("%s row %d has %d columns, need id and label", path, n+2, len(r))
		}
		l := domain.ClaimLabel(strings.ToLower(strings.TrimSpace(r[lblCol])))
		switch l {
		case domain.LabelSupported, domain.LabelPartial, domain.LabelContradicted, domain.LabelAbsent:
			out[strings.TrimSpace(r[idCol])] = l
		default:
			return nil, fmt.Errorf("%s: id %q has label %q, labels must be complete", path, r[idCol], r[lblCol])
		}
	}
	return out, nil
}

// worst collapses per-claim results into one explanation label.
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

type row struct {
	ID, ConceptID string
	Gold, Pred    domain.ClaimLabel
}

type stats struct {
	Model           string
	Rows            []row
	Proposed, Valid int
	Calls           []llm.Call
	Models          llm.Models // prices come from internal/llm/models.json
}

var labelOrder = []domain.ClaimLabel{domain.LabelSupported, domain.LabelPartial, domain.LabelContradicted, domain.LabelAbsent}

// report renders the results file. It states numbers only and names no winner.
func (s stats) report(now time.Time) string {
	var wrongN, wrongHit, okN, okFalse int
	conf := map[domain.ClaimLabel]map[domain.ClaimLabel]int{}
	for _, r := range s.Rows {
		if conf[r.Gold] == nil {
			conf[r.Gold] = map[domain.ClaimLabel]int{}
		}
		conf[r.Gold][r.Pred]++
		switch r.Gold {
		case domain.LabelContradicted, domain.LabelAbsent:
			wrongN++
			if r.Pred == domain.LabelContradicted || r.Pred == domain.LabelAbsent {
				wrongHit++
			}
		case domain.LabelSupported:
			okN++
			if r.Pred == domain.LabelContradicted {
				okFalse++
			}
		}
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# Checker eval: %s\n\nRun %s UTC. %d explanations, each treated as a finished session (absent allowed).\nExplanation label = worst claim label, order contradicted > absent > partial > supported.\n\n",
		s.Model, now.UTC().Format("2006-01-02 15:04"), len(s.Rows))
	b.WriteString("## Gate numbers\n\n")
	fmt.Fprintf(&b, "- Wrong explanations flagged contradicted or absent: %d of %d\n", wrongHit, wrongN)
	fmt.Fprintf(&b, "- Correct explanations falsely flagged contradicted: %d of %d\n", okFalse, okN)
	if s.Proposed == 0 {
		b.WriteString("- Corrections with a valid citation: n/a (the model proposed no corrections)\n")
	} else {
		fmt.Fprintf(&b, "- Corrections with a valid citation: %.1f%% (%d of %d)\n", 100*float64(s.Valid)/float64(s.Proposed), s.Valid, s.Proposed)
	}
	b.WriteString("\n## Confusion table\n\nRows are the human label, columns are the checker label.\n\n| human \\ checker |")
	for _, l := range labelOrder {
		fmt.Fprintf(&b, " %s |", l)
	}
	b.WriteString("\n|---|---|---|---|---|\n")
	for _, g := range labelOrder {
		fmt.Fprintf(&b, "| %s |", g)
		for _, p := range labelOrder {
			fmt.Fprintf(&b, " %d |", conf[g][p])
		}
		b.WriteString("\n")
	}
	b.WriteString("\n## Cost and latency per call\n\n")
	b.WriteString(s.callTable())
	b.WriteString("\n## Per explanation\n\n| id | concept | human | checker |\n|---|---|---|---|\n")
	for _, r := range s.Rows {
		fmt.Fprintf(&b, "| %s | %s | %s | %s |\n", r.ID, r.ConceptID, r.Gold, r.Pred)
	}
	return b.String()
}

func (s stats) callTable() string {
	n := len(s.Calls)
	if n == 0 {
		return "No calls.\n"
	}
	lat := make([]time.Duration, n)
	var in, out int64
	var total float64
	var sum time.Duration
	for i, c := range s.Calls {
		lat[i] = c.Latency
		sum += c.Latency
		in += c.In
		out += c.Out
		total += c.Cost
	}
	sort.Slice(lat, func(i, j int) bool { return lat[i] < lat[j] })
	var b strings.Builder
	fmt.Fprintf(&b, "- Calls: %d\n- Latency: mean %s, median %s, max %s\n", n, (sum / time.Duration(n)).Round(time.Millisecond), lat[n/2].Round(time.Millisecond), lat[n-1].Round(time.Millisecond))
	fmt.Fprintf(&b, "- Tokens per call: mean %d in, %d out\n", in/int64(n), out/int64(n))
	if !s.Models.Priced(s.Model) {
		fmt.Fprintf(&b, "- Cost: not computed, %s has no price in internal/llm/models.json\n", s.Model)
	} else {
		fmt.Fprintf(&b, "- Cost: $%.5f per call, $%.4f total (prices from models.json)\n", total/float64(n), total)
	}
	return b.String()
}
