package planner

import (
	"reflect"
	"testing"
	"time"

	"github.com/CrisH2307/repetition-study/internal/domain"
)

func tp(s string) *time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return &t
}

func state(tz string, cs ...domain.ConceptState) domain.StudyState {
	return domain.StudyState{User: domain.User{TimeZone: tz}, Courses: []domain.CourseState{{
		Subject: domain.Subject{ID: "s1", Name: "Calc", WeeklyHours: 3}, Concepts: cs}}}
}

func TestTodayQueue(t *testing.T) {
	// Toronto, Wed 2026-11-18 20:00 local (Thu 01:00 UTC).
	now := *tp("2026-11-19T01:00:00Z")
	st := state("America/Toronto",
		domain.ConceptState{ConceptID: "later", Name: "Later", DueAt: tp("2026-11-20T15:00:00Z"), Taught: true},
		// 23:30 local Wed is still today even though it is Thu in UTC.
		domain.ConceptState{ConceptID: "late", Name: "Late", DueAt: tp("2026-11-19T04:30:00Z"), Taught: true},
		domain.ConceptState{ConceptID: "old", Name: "Old", DueAt: tp("2026-11-16T15:00:00Z"), Taught: true},
		domain.ConceptState{ConceptID: "new", Name: "New"},
		// 00:30 local Thu is tomorrow although still Wed... in UTC it is 05:30 Thu.
		domain.ConceptState{ConceptID: "tom", Name: "Tom", DueAt: tp("2026-11-19T05:30:00Z"), Taught: true},
	)
	got, err := Today{}.Today(st, now)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Date.Equal(*tp("2026-11-18T05:00:00Z")) {
		t.Fatalf("date %v", got.Date)
	}
	var ids, reasons []string
	for _, it := range got.Courses[0].Items {
		ids, reasons = append(ids, it.ConceptID), append(reasons, it.Reason)
	}
	if want := []string{"old", "new", "late"}; !reflect.DeepEqual(ids, want) {
		// new has no due date: shown at local midnight today, after old, before late.
		t.Fatalf("ids %v want %v", ids, want)
	}
	if want := []string{"Overdue by 2 days", "Not taught yet", "Due today"}; !reflect.DeepEqual(reasons, want) {
		t.Fatalf("reasons %v", reasons)
	}
	if got.Courses[0].Minutes != 5+15+5 {
		t.Fatalf("minutes %d", got.Courses[0].Minutes)
	}
}

func TestWeekDSTToronto(t *testing.T) {
	// Fall back Sun 2026-11-01: week of Mon 2026-10-26 has a 25 h Sunday.
	now := *tp("2026-10-28T14:00:00Z") // Wed 10:00 local (EDT)
	ws := *tp("2026-10-26T04:00:00Z")
	st := state("America/Toronto",
		domain.ConceptState{ConceptID: "a", Name: "A", DueAt: tp("2026-10-27T14:00:00Z"), Taught: true},
		// 23:30 EST Sat 10-31 is 03:30 UTC Sun... still Saturday local (EDT: 03:30Z).
		domain.ConceptState{ConceptID: "b", Name: "B", DueAt: tp("2026-11-01T04:30:00Z"), Taught: true}, // Sun 00:30 EDT
		domain.ConceptState{ConceptID: "c", Name: "C", DueAt: tp("2026-11-02T04:30:00Z"), Taught: true}, // Sun 23:30 EST
		domain.ConceptState{ConceptID: "d", Name: "D", DueAt: tp("2026-11-02T05:30:00Z"), Taught: true}, // Mon next week
	)
	st.Courses[0].Events = []domain.ReviewEvent{
		{EventType: domain.EventTeachBack, CreatedAt: *tp("2026-10-26T15:00:00Z")},
		{EventType: domain.EventCardReview, CreatedAt: *tp("2026-11-02T04:59:00Z")},
		{EventType: domain.EventCardReview, CreatedAt: *tp("2026-11-02T05:00:00Z")}, // next week
	}
	w, err := Today{}.Week(st, ws, now)
	if err != nil {
		t.Fatal(err)
	}
	wantStarts := []string{"2026-10-26T04:00:00Z", "2026-10-27T04:00:00Z", "2026-10-28T04:00:00Z", "2026-10-29T04:00:00Z",
		"2026-10-30T04:00:00Z", "2026-10-31T04:00:00Z", "2026-11-01T04:00:00Z"}
	var counts []int
	for i, d := range w.Days {
		if !d.Date.Equal(*tp(wantStarts[i])) {
			t.Fatalf("day %d starts %v", i, d.Date)
		}
		counts = append(counts, len(d.Items))
	}
	// a was due Tue (overdue vs Wed): moves to today Wed. b Sunday, c Sunday.
	if want := []int{0, 0, 1, 0, 0, 0, 2}; !reflect.DeepEqual(counts, want) {
		t.Fatalf("counts %v want %v", counts, want)
	}
	h := w.Courses[0]
	if h.HoursUsed != 20.0/60 { // 15 + 5; the event on the next local Monday is out
		t.Fatalf("used %v", h.HoursUsed)
	}
	if h.HoursBudget != 3 || h.Reason != "" {
		t.Fatalf("budget %v %q", h.HoursBudget, h.Reason)
	}
	if len(w.Notes) != 1 || w.Notes[0] != "1 overdue topic is moved to Wed, Oct 28" {
		t.Fatalf("notes %v", w.Notes)
	}
}

func TestBudgetRaiseAndFutureWeek(t *testing.T) {
	var cs []domain.ConceptState
	for i := 0; i < 14; i++ { // 14 never-taught topics = 210 min = 3.5 h
		cs = append(cs, domain.ConceptState{ConceptID: string(rune('a' + i)), Name: string(rune('a' + i))})
	}
	st := state("UTC", cs...)
	st.Courses[0].Subject.WeeklyHours = 2
	got, _ := Today{}.Week(st, *tp("2026-11-16T00:00:00Z"), *tp("2026-11-16T09:00:00Z"))
	h := got.Courses[0]
	if h.HoursBudget != 3.5 || h.Reason != "Up 1.5 h: 14 topics due this week" {
		t.Fatalf("%v %q", h.HoursBudget, h.Reason)
	}
	// Next week: never-taught topics sit on its first day.
	nxt, _ := Today{}.Week(st, *tp("2026-11-23T00:00:00Z"), *tp("2026-11-16T09:00:00Z"))
	if len(nxt.Days[0].Items) != 14 {
		t.Fatalf("day0 %d", len(nxt.Days[0].Items))
	}
	// Past week: nothing is placed.
	past, _ := Today{}.Week(st, *tp("2026-11-09T00:00:00Z"), *tp("2026-11-16T09:00:00Z"))
	for _, d := range past.Days {
		if len(d.Items) != 0 {
			t.Fatal("past week has items")
		}
	}
}

func TestBadZoneAndDeterminism(t *testing.T) {
	if _, err := (Today{}).Today(state("Mars/Base"), time.Now()); err == nil {
		t.Fatal("want error")
	}
	st := state("America/Toronto", domain.ConceptState{ConceptID: "x", Name: "X"})
	a, _ := Today{}.Today(st, *tp("2026-11-16T15:00:00Z"))
	b, _ := Today{}.Today(st, *tp("2026-11-16T15:00:00Z"))
	if !reflect.DeepEqual(a, b) {
		t.Fatal("not deterministic")
	}
}
