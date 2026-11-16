package metrics

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/CrisH2307/repetition-study/internal/domain"
	"github.com/CrisH2307/repetition-study/internal/fakes"
)

func d(s string) time.Time { t, _ := time.Parse("2006-01-02", s); return t }

func TestCompute(t *testing.T) {
	st := &fakes.MetricsStore{
		Weeks: []domain.WeekCounts{
			{WeekStart: d("2026-10-05"), ActiveUsers: 2, Sessions: 5, CheckedSessions: 4, UncheckedSessions: 1, Disputes: 2},
			{WeekStart: d("2026-10-12")}, // empty week: no divide by zero
		},
		Users: []domain.UserWeeks{
			{UserID: "u1", JoinedAt: d("2026-10-07"), ActiveWeeks: []time.Time{d("2026-10-05"), d("2026-11-02")}}, // kept
			{UserID: "u2", JoinedAt: d("2026-10-06"), ActiveWeeks: []time.Time{d("2026-10-05")}},                  // lost
			{UserID: "u3", JoinedAt: d("2026-11-02"), ActiveWeeks: []time.Time{d("2026-11-02")}},                  // too new
		},
	}
	r, err := Compute(context.Background(), st, d("2026-10-05"), d("2026-10-19"), d("2026-11-16"))
	if err != nil {
		t.Fatal(err)
	}
	w := r.Weeks[0]
	if w.SessionsPerUser != 2.5 || w.DisputesPerChecked != 0.5 || w.UncheckedRate != 0.2 {
		t.Fatalf("week row %+v", w)
	}
	if z := r.Weeks[1]; z.SessionsPerUser != 0 || z.UncheckedRate != 0 {
		t.Fatalf("empty week %+v", z)
	}
	if r.Week4Eligible != 2 || r.Week4Retention != 0.5 {
		t.Fatalf("retention %v of %d", r.Week4Retention, r.Week4Eligible)
	}
}

func TestNoEligibleUsers(t *testing.T) {
	r, _ := Compute(context.Background(), &fakes.MetricsStore{}, d("2026-10-05"), d("2026-10-12"), d("2026-11-16"))
	if r.Week4Retention != -1 {
		t.Fatal("want -1")
	}
}

func TestCSVHidesIDs(t *testing.T) {
	us := []domain.UserWeeks{{UserID: "secret-user-id", JoinedAt: d("2026-10-07")}}
	var b bytes.Buffer
	if err := WriteCSV(&b, Report{Week4Retention: -1}, us, "salt"); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(b.String(), "secret-user-id") || !strings.Contains(b.String(), HashID("salt", "secret-user-id")) || !strings.Contains(b.String(), "n/a") {
		t.Fatal(b.String())
	}
}

func TestMonday(t *testing.T) {
	if got := Monday(d("2026-11-15")); !got.Equal(d("2026-11-09")) { // Sunday
		t.Fatal(got)
	}
}
