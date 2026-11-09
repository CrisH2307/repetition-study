// Package srs is the scheduler. Code owns every date; FSRS (go-fsrs v4,
// default FSRS-6 weights) is the only memory model. This file is pure logic
// with no database, so it is unit tested and benchmarked on its own.
package srs

import (
	"errors"
	"time"

	fsrs "github.com/open-spaced-repetition/go-fsrs/v4"

	"github.com/CrisH2307/repetition-study/internal/domain"
)

// newFSRS uses default weights and retention. Short-term learning steps are off
// because they put a first Good review 10 minutes out, which is meaningless in
// a weekly study plan. All intervals are whole days.
func newFSRS() *fsrs.FSRS {
	p := fsrs.DefaultParam()
	p.EnableShortTerm = false
	return fsrs.NewFSRS(p)
}

func rating(g domain.Grade) (fsrs.Rating, error) {
	switch g {
	case domain.GradeAgain:
		return fsrs.Again, nil
	case domain.GradeHard:
		return fsrs.Hard, nil
	case domain.GradeGood:
		return fsrs.Good, nil
	case domain.GradeEasy:
		return fsrs.Easy, nil
	}
	return 0, errors.New("srs: unknown grade " + string(g))
}

// teachBackGrade is the worst result across claims that count: contradicted or
// absent gives Again, partial gives Hard, all supported gives Good. Never Easy.
// Claims in notRequired are skipped. Results with no claim, or a claim we do
// not know, count (a wrong statement should not be ignored). No countable
// results is an error, not a silent Good.
func teachBackGrade(results []domain.ClaimResult, notRequired map[string]bool) (domain.Grade, error) {
	g, n := domain.GradeGood, 0
	for _, r := range results {
		if r.ClaimID != nil && notRequired[*r.ClaimID] {
			continue
		}
		n++
		switch r.Label {
		case domain.LabelContradicted, domain.LabelAbsent:
			g = domain.GradeAgain
		case domain.LabelPartial:
			if g != domain.GradeAgain {
				g = domain.GradeHard
			}
		case domain.LabelSupported:
		default:
			return "", errors.New("srs: unknown label " + string(r.Label))
		}
	}
	if n == 0 {
		return "", errors.New("srs: no required claim results to grade")
	}
	return g, nil
}

// grade applies one review to a concept's FSRS card.
func grade(f *fsrs.FSRS, c fsrs.Card, now time.Time, g domain.Grade) (fsrs.Card, error) {
	r, err := rating(g)
	if err != nil {
		return c, err
	}
	info, err := f.Next(c, now, r)
	if err != nil {
		return c, err
	}
	return info.Card, nil
}

// interleave orders items so two from the same concept are never adjacent
// while another concept still has items left. It takes one item per concept per
// round, concepts in order of first appearance. Order within a concept is kept.
func interleave[T any](items []T, concept func(T) string) []T {
	var order []string
	by := map[string][]T{}
	for _, it := range items {
		k := concept(it)
		if _, ok := by[k]; !ok {
			order = append(order, k)
		}
		by[k] = append(by[k], it)
	}
	out := make([]T, 0, len(items))
	for round := 0; len(out) < len(items); round++ {
		for _, k := range order {
			if round < len(by[k]) {
				out = append(out, by[k][round])
			}
		}
	}
	return out
}
