// Package cite is the one place that verifies a citation quote against an
// approved source. Nothing else in the repo re-checks quotes.
package cite

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// MinQuote is the shortest quote, in characters after Normalize, that can back
// a correction. A shorter quote passes only if it is a whole display formula.
const MinQuote = 20

// Normalize collapses every whitespace run to one space and trims the ends.
// No case folding and no quote-mark mapping: everything else must match exactly.
func Normalize(s string) string { return strings.Join(strings.Fields(s), " ") }

// Verify reports whether quote is an exact substring of sourceText after
// Normalize, and is long enough to mean something.
func Verify(sourceText, quote string) bool {
	q := Normalize(quote)
	if q == "" || !strings.Contains(Normalize(sourceText), q) {
		return false
	}
	return utf8.RuneCountInString(q) >= MinQuote || wholeFormula(sourceText, q)
}

// wholeFormula reports whether q, normalized, is one entire display formula
// block of src, with or without its delimiters.
func wholeFormula(src, q string) bool {
	for _, d := range [][2]string{{"$$", "$$"}, {`\[`, `\]`}} {
		for _, b := range displayBlocks(src, d[0], d[1]) {
			inner := Normalize(b[len(d[0]) : len(b)-len(d[1])])
			if inner != "" && (q == inner || q == Normalize(b)) {
				return true
			}
		}
	}
	return false
}

// displayBlocks returns each open...close span whose opener has whitespace or
// the start of text before it and whose closer has whitespace or the end of
// text after it. So $x$$y$ and \\[2pt] never open a block.
func displayBlocks(src, open, close string) []string {
	var out []string
	for i := 0; ; {
		k := strings.Index(src[i:], open)
		if k < 0 {
			return out
		}
		start := i + k
		i = start + len(open)
		if !edgeBefore(src, start) {
			continue
		}
		for j := i; ; {
			m := strings.Index(src[j:], close)
			if m < 0 {
				return out
			}
			end := j + m + len(close)
			if edgeAfter(src, end) {
				out = append(out, src[start:end])
				i = end
				break
			}
			j = end
		}
	}
}

func edgeBefore(s string, i int) bool {
	r, _ := utf8.DecodeLastRuneInString(s[:i])
	return i == 0 || unicode.IsSpace(r)
}

func edgeAfter(s string, i int) bool {
	r, _ := utf8.DecodeRuneInString(s[i:])
	return i == len(s) || unicode.IsSpace(r)
}
