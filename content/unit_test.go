// Package content validates the pack in this folder against internal/domain.
package content

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/CrisH2307/repetition-study/internal/domain"
)

type unit struct {
	Subject  domain.Subject   `json:"subject"`
	Sources  []domain.Source  `json:"sources"`
	Concepts []domain.Concept `json:"concepts"`
}

// allowlistEntry fixes the fields of content/allowlist.json. It is [] today.
type allowlistEntry struct {
	URL        string `json:"url"`
	Publisher  string `json:"publisher"`
	License    string `json:"license"`
	ReviewedBy string `json:"reviewed_by"`
	ReviewedAt string `json:"reviewed_at"`
}

func strict(t *testing.T, path string, v any) {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		t.Fatalf("%s: %v", path, err)
	}
}

// normalize mirrors scripts/parse_pdf.py.
func normalize(s string) string {
	var out []string
	for _, l := range strings.Split(s, "\n") {
		if f := strings.Join(strings.Fields(l), " "); f != "" {
			out = append(out, f)
		}
	}
	return strings.Join(out, "\n")
}

var (
	uuidRe    = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	locatorRe = regexp.MustCompile(`^p\.(\d+), (Ex 3\.\d+|Q\d|Notes \d)( \(cont\.\))?$`)
)

// page returns the text of one page of the source, or false when it is missing.
func page(text string, n int) (string, bool) {
	_, rest, ok := strings.Cut(text, "<!-- page "+strconv.Itoa(n)+" -->\n")
	if !ok {
		return "", false
	}
	body, _, _ := strings.Cut(rest, "<!-- page ")
	return body, true
}

// TestFixture always runs: the original fixture is committed.
func TestFixture(t *testing.T) { checkUnit(t, "fixture/unit.json", "fixture/notes.md") }

// TestPrivateUnit checks the real course unit. It skips when the private copy is absent (CI).
func TestPrivateUnit(t *testing.T) {
	if _, err := os.Stat("private/source.md"); err != nil {
		t.Skip("content/private/source.md not present; see content/README.md")
	}
	checkUnit(t, "private/unit.json", "private/source.md")
}

func checkUnit(t *testing.T, unitPath, sourcePath string) {
	var u unit
	strict(t, unitPath, &u)
	var al []allowlistEntry
	strict(t, "allowlist.json", &al)
	if len(al) != 0 {
		t.Fatal("allowlist must stay empty until licenses are reviewed")
	}

	if len(u.Sources) != 1 || u.Sources[0].Kind != domain.SourceCourse {
		t.Fatal("want exactly one course source")
	}
	src := u.Sources[0]
	raw, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	if src.Text != normalize(string(raw)) {
		t.Fatal("unit.json source text is stale: run python3 scripts/parse_pdf.py --sync content/private/source.md content/private/unit.json")
	}

	if n := len(u.Concepts); n == 0 || n > 8 {
		t.Fatalf("want 1 to 8 concepts, got %d", n)
	}
	seen := map[string]bool{}
	for _, id := range []string{u.Subject.ID, src.ID} {
		seen[id] = true
	}
	for _, c := range u.Concepts {
		if c.SubjectID != u.Subject.ID || !uuidRe.MatchString(c.ID) || seen[c.ID] {
			t.Errorf("concept %q: bad or duplicate id or subject", c.Name)
		}
		seen[c.ID] = true
		required := 0
		for _, cl := range c.Claims {
			if !uuidRe.MatchString(cl.ID) || seen[cl.ID] {
				t.Errorf("claim %s: bad or duplicate id", cl.ID)
			}
			seen[cl.ID] = true
			if cl.ConceptID != c.ID || cl.SourceID != src.ID || strings.TrimSpace(cl.Text) == "" {
				t.Errorf("claim %s: wrong concept, source or empty text", cl.ID)
			}
			if cl.Required {
				required++
			}
			m := locatorRe.FindStringSubmatch(cl.Locator)
			if m == nil {
				t.Errorf("claim %s: locator %q not in the form 'p.N, Ex 3.x' or 'p.N, Qn'", cl.ID, cl.Locator)
				continue
			}
			n, _ := strconv.Atoi(m[1])
			body, ok := page(src.Text, n)
			if !ok {
				t.Errorf("claim %s: page %d not in source", cl.ID, n)
				continue
			}
			if m[3] != "" || strings.HasPrefix(m[2], "Notes") { // continuation page, or fixture: no problem heading to find
				continue
			}
			var re *regexp.Regexp
			if strings.HasPrefix(m[2], "Q") {
				re = regexp.MustCompile(`(?m)^` + m[2][1:] + `\. `)
			} else {
				re = regexp.MustCompile(`Ex(ercise|\.)? ` + regexp.QuoteMeta(m[2][3:]) + `\b`)
			}
			if !re.MatchString(body) {
				t.Errorf("claim %s: %q does not resolve on page %d", cl.ID, cl.Locator, n)
			}
		}
		if required < 3 || required > 5 {
			t.Errorf("concept %q: want 3 to 5 required claims, got %d", c.Name, required)
		}
	}
}

// TestSeedPack checks the generated SQL is current. It skips without the private copy.
func TestSeedPack(t *testing.T) {
	pack, err := os.ReadFile("private/seed_pack.sql")
	if err != nil {
		t.Skip("content/private/seed_pack.sql not present; see content/README.md")
	}
	var u unit
	strict(t, "private/unit.json", &u)
	if !strings.Contains(string(pack), u.Sources[0].Text) {
		t.Error("seed_pack.sql is stale: run python3 scripts/parse_pdf.py --sql content/private/unit.json content/private/seed_pack.sql")
	}
	for _, c := range u.Concepts {
		for _, cl := range c.Claims {
			if !strings.Contains(string(pack), cl.ID) {
				t.Errorf("seed_pack.sql lacks claim %s: regenerate it", cl.ID)
			}
		}
	}
}

// TestEval checks the eval files line up with the concepts in unit.json (no source text needed).
func TestEval(t *testing.T) {
	var u unit
	strict(t, "unit.json", &u)
	concepts := map[string]bool{}
	for _, c := range u.Concepts {
		concepts[c.ID] = true
	}

	f, err := os.Open("../eval/data/explanations.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	ids := map[string]bool{}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		var e struct {
			ID        string `json:"id"`
			ConceptID string `json:"concept_id"`
			Text      string `json:"text"`
		}
		d := json.NewDecoder(strings.NewReader(sc.Text()))
		d.DisallowUnknownFields() // a stray "label" key would leak the answer
		if err := d.Decode(&e); err != nil {
			t.Fatal(err)
		}
		if e.ID == "" || ids[e.ID] || !concepts[e.ConceptID] || strings.TrimSpace(e.Text) == "" {
			t.Errorf("explanation %q: empty or duplicate id, unknown concept or empty text", e.ID)
		}
		ids[e.ID] = true
	}
	if len(ids) != 20 {
		t.Errorf("want 20 explanations, got %d", len(ids))
	}
	for _, name := range []string{"labels_cris.csv", "labels_dino.csv"} {
		b, err := os.ReadFile("../eval/data/" + name)
		if err != nil {
			t.Fatal(err)
		}
		rows, err := csv.NewReader(bytes.NewReader(b)).ReadAll()
		if err != nil || len(rows) != 21 || rows[0][0] != "id" || rows[0][1] != "label" {
			t.Errorf("%s: want header id,label and 20 rows (err %v)", name, err)
			continue
		}
		for _, r := range rows[1:] {
			if !ids[r[0]] {
				t.Errorf("%s: unknown id %q", name, r[0])
			}
		}
	}
}
