// Package content validates the pack in this folder against internal/domain.
package content

import (
	"bytes"
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
	locatorRe = regexp.MustCompile(`^p\.(\d+), (Ex 3\.\d+|Q\d)( \(cont\.\))?$`)
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

func TestUnit(t *testing.T) {
	var u unit
	strict(t, "unit.json", &u)
	var al []allowlistEntry
	strict(t, "allowlist.json", &al)
	if len(al) != 0 {
		t.Fatal("allowlist must stay empty until licenses are reviewed")
	}

	if len(u.Sources) != 1 || u.Sources[0].Kind != domain.SourceCourse {
		t.Fatal("want exactly one course source")
	}
	src := u.Sources[0]
	raw, err := os.ReadFile("source.md")
	if err != nil {
		t.Fatal(err)
	}
	if src.Text != normalize(string(raw)) {
		t.Fatal("unit.json source text is stale: run python3 scripts/parse_pdf.py --sync content/source.md content/unit.json")
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
			if m[3] != "" { // continuation page: the problem heading is on an earlier page
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
