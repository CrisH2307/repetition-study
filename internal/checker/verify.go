package checker

import (
	"strings"
	"unicode"

	"github.com/CrisH2307/repetition-study/internal/domain"
)

// hintRun is the longest run of consecutive words a probe_hint may share with a
// source. A hint with a run of this length or more is rejected.
// Unverified choice, the prompt says only "a long run of words".
const hintRun = 5

// Drop records one thing code removed from the model output.
type Drop struct {
	ClaimID string // empty when the result had no claim id
	Reason  string
}

// Outcome is Check plus the numbers cmd/evalcheck reports.
type Outcome struct {
	Results []domain.ClaimResult
	Drops   []Drop
	// Proposed counts contradicted or partial results where the model offered a
	// correction or citation. Valid counts those whose citation passed.
	Proposed, Valid int
}

// normalize collapses all whitespace runs to one space.
func normalize(s string) string { return strings.Join(strings.Fields(s), " ") }

// verifyCitation reports whether c quotes an approved source exactly, after
// whitespace normalization only. No case folding, no quote-mark mapping.
func verifyCitation(c *domain.Citation, sources map[string]string) bool {
	if c == nil {
		return false
	}
	text, ok := sources[c.SourceID]
	if !ok {
		return false
	}
	q := normalize(c.Quote)
	return q != "" && strings.Contains(normalize(text), q)
}

func words(s string) []string {
	return strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
}

// hintLeaks reports whether hint shares hintRun or more consecutive words with
// any source text.
func hintLeaks(hint string, sources map[string]string) bool {
	hw := words(hint)
	if len(hw) < hintRun {
		return false
	}
	grams := map[string]bool{}
	for i := 0; i+hintRun <= len(hw); i++ {
		grams[strings.Join(hw[i:i+hintRun], " ")] = true
	}
	for _, t := range sources {
		sw := words(t)
		for i := 0; i+hintRun <= len(sw); i++ {
			if grams[strings.Join(sw[i:i+hintRun], " ")] {
				return true
			}
		}
	}
	return false
}

func approved(s domain.Source) bool {
	switch s.Kind {
	case domain.SourceCourse:
		return true
	case domain.SourceAllowlist:
		return s.ReviewedBy != "" && s.ReviewedAt != nil
	}
	return false
}

func approvedSources(in []domain.Source) map[string]string {
	m := map[string]string{}
	for _, s := range in {
		if approved(s) {
			m[s.ID] = s.Text
		}
	}
	return m
}

func validLabel(l domain.ClaimLabel) bool {
	switch l {
	case domain.LabelSupported, domain.LabelPartial, domain.LabelContradicted, domain.LabelAbsent:
		return true
	}
	return false
}

// enforce applies every code rule to the model output. It never adds a
// correction, it only removes what cannot be proven.
func enforce(concept domain.Concept, sources map[string]string, raw []domain.ClaimResult, allowAbsent bool) Outcome {
	var out Outcome
	known := map[string]bool{}
	for _, c := range concept.Claims {
		known[c.ID] = true
	}
	drop := func(r domain.ClaimResult, why string) {
		id := ""
		if r.ClaimID != nil {
			id = *r.ClaimID
		}
		out.Drops = append(out.Drops, Drop{ClaimID: id, Reason: why})
	}
	seen := map[string]bool{}
	for _, r := range raw {
		if !validLabel(r.Label) {
			drop(r, "invalid label "+string(r.Label))
			continue
		}
		if r.Label == domain.LabelAbsent && !allowAbsent {
			drop(r, "absent is not allowed during a session")
			continue
		}
		if r.ClaimID != nil && !known[*r.ClaimID] {
			drop(r, "unknown claim id, treated as an unmatched statement")
			r.ClaimID = nil
		}
		if r.ClaimID != nil {
			seen[*r.ClaimID] = true
		}

		proposed := (r.Label == domain.LabelContradicted || r.Label == domain.LabelPartial) &&
			(r.Correction != nil || r.Citation != nil)
		if proposed {
			out.Proposed++
		}
		switch {
		case r.Label == domain.LabelSupported || r.Label == domain.LabelAbsent:
			// Nothing to correct.
			r.Correction, r.Citation = nil, nil
		case verifyCitation(r.Citation, sources) && r.Correction != nil && strings.TrimSpace(*r.Correction) != "":
			if proposed {
				out.Valid++
			}
		default:
			if r.Correction != nil || r.Citation != nil {
				drop(r, "citation failed verification, correction removed")
			}
			r.Correction, r.Citation = nil, nil
			if r.Label == domain.LabelContradicted {
				r.Label = domain.LabelPartial
			}
		}
		if r.ProbeHint != nil && hintLeaks(*r.ProbeHint, sources) {
			drop(r, "probe_hint shares a long run of words with a source, removed")
			r.ProbeHint = nil
		}
		out.Results = append(out.Results, r)
	}
	if allowAbsent {
		for _, c := range concept.Claims {
			if c.Required && !seen[c.ID] {
				id := c.ID
				out.Results = append(out.Results, domain.ClaimResult{ClaimID: &id, StudentText: "", Label: domain.LabelAbsent})
			}
		}
	}
	return out
}
