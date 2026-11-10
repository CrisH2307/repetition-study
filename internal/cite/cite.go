// Package cite is the one place that verifies a citation quote against an
// approved source. Nothing else in the repo re-checks quotes.
package cite

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

// MinQuote is the shortest quote, in characters after Normalize, that can back
// a correction. A shorter quote passes only if it is a whole display formula.
const MinQuote = 20

// displayMath matches $$...$$ and \[...\] blocks.
var displayMath = regexp.MustCompile(`(?s)\$\$(.+?)\$\$|\\\[(.+?)\\\]`)

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
	for _, m := range displayMath.FindAllStringSubmatch(src, -1) {
		inner := Normalize(m[1] + m[2])
		if inner != "" && (q == inner || q == Normalize(m[0])) {
			return true
		}
	}
	return false
}
