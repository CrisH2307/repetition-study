package srs

import (
	"testing"
	"time"

	fsrs "github.com/open-spaced-repetition/go-fsrs/v4"

	"github.com/CrisH2307/repetition-study/internal/domain"
	"github.com/CrisH2307/repetition-study/internal/fakes"
)

func res(l domain.ClaimLabel) domain.ClaimResult { return domain.ClaimResult{Label: l} }

func TestTeachBackGrade(t *testing.T) {
	optional := "opt"
	skipped := domain.ClaimResult{ClaimID: &optional, Label: domain.LabelContradicted}
	cases := []struct {
		name string
		in   []domain.ClaimResult
		want domain.Grade
	}{
		{"all supported is good", []domain.ClaimResult{res("supported"), res("supported")}, domain.GradeGood},
		{"partial is hard", []domain.ClaimResult{res("supported"), res("partial")}, domain.GradeHard},
		{"contradicted is again", []domain.ClaimResult{res("supported"), res("contradicted")}, domain.GradeAgain},
		{"absent is again", []domain.ClaimResult{res("absent"), res("supported")}, domain.GradeAgain},
		{"again beats partial", []domain.ClaimResult{res("partial"), res("contradicted")}, domain.GradeAgain},
		{"non-required claim is ignored", []domain.ClaimResult{res("supported"), skipped}, domain.GradeGood},
	}
	for _, c := range cases {
		got, err := teachBackGrade(c.in, map[string]bool{optional: true})
		if err != nil || got != c.want {
			t.Errorf("%s: got %q, %v want %q", c.name, got, err, c.want)
		}
		if got == domain.GradeEasy {
			t.Errorf("%s: teach-back must never give easy", c.name)
		}
	}
	if _, err := teachBackGrade(nil, nil); err == nil {
		t.Error("no results should be an error")
	}
	if _, err := teachBackGrade([]domain.ClaimResult{res("bogus")}, nil); err == nil {
		t.Error("unknown label should be an error")
	}
}

// A concept due in 7 days must come back earlier after a contradicted teach-back.
func TestContradictedPullsDueEarlier(t *testing.T) {
	f, now := newFSRS(), fakes.Epoch
	c := fsrs.NewCard(now)
	var err error
	for c.Due.Sub(now) < 7*24*time.Hour {
		now = c.Due
		if c, err = grade(f, c, now, domain.GradeGood); err != nil {
			t.Fatal(err)
		}
	}
	before := c.Due
	after, err := grade(f, c, now, domain.GradeAgain)
	if err != nil {
		t.Fatal(err)
	}
	if !after.Due.Before(before) {
		t.Fatalf("due %v should be before %v", after.Due, before)
	}
}

func TestInterleave(t *testing.T) {
	type it struct {
		c string
		n int
	}
	in := []it{{"A", 1}, {"B", 1}, {"C", 1}, {"A", 2}, {"A", 3}, {"B", 2}}
	out := interleave(in, func(x it) string { return x.c })
	if len(out) != len(in) {
		t.Fatal("lost items")
	}
	for i := 1; i < len(out); i++ {
		if out[i].c != out[i-1].c {
			continue
		}
		// adjacent same concept is only allowed when no other concept is left
		for _, rest := range out[i:] {
			if rest.c != out[i].c {
				t.Fatalf("same concept twice in a row at %d while %s remains: %v", i, rest.c, out)
			}
		}
	}
	if out[0].c != "A" || out[1].c != "B" || out[2].c != "C" {
		t.Fatalf("round one should take one per concept: %v", out)
	}
}

// 1,000 concepts: grade each and order the due list. Must stay under 20 ms.
func BenchmarkSchedule1000(b *testing.B) {
	f, now := newFSRS(), fakes.Epoch
	cards := make([]fsrs.Card, 1000)
	for i := range cards {
		cards[i] = fsrs.NewCard(now)
		cards[i], _ = grade(f, cards[i], now, domain.GradeGood)
	}
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		start := time.Now()
		items := make([]int, 0, len(cards))
		for i, c := range cards {
			c, err := grade(f, c, c.Due, domain.GradeGood)
			if err != nil {
				b.Fatal(err)
			}
			cards[i] = c
			items = append(items, i%100)
		}
		_ = interleave(items, func(i int) string { return string(rune(i)) })
		if d := time.Since(start); d > 20*time.Millisecond {
			b.Fatalf("1000 concepts took %v, limit 20ms", d)
		}
		for i := range cards { // reset so every iteration does the same work
			cards[i], _ = grade(f, fsrs.NewCard(now), now, domain.GradeGood)
		}
	}
}
