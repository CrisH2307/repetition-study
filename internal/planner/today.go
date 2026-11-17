package planner

import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/CrisH2307/repetition-study/internal/domain"
)

// Today and Week are pure code over StudyState (D31, D13): no timed blocks,
// fixed minute estimates, and a reason in words for every number.
const (
	teachMinutes  = 15
	recallMinutes = 5
)

// Today implements domain.TodayPlanner.
type Today struct{}

var _ domain.TodayPlanner = Today{}

func userLoc(u domain.User) (*time.Location, error) {
	if u.TimeZone == "" {
		return time.UTC, nil
	}
	loc, err := time.LoadLocation(u.TimeZone)
	if err != nil {
		return nil, fmt.Errorf("planner: bad time zone %q: %w", u.TimeZone, err)
	}
	return loc, nil
}

// civil is a local calendar day, kept as UTC midnight so day maths ignores DST.
func civil(t time.Time, loc *time.Location) time.Time {
	y, m, d := t.In(loc).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// localMidnight is the instant a civil day starts in loc, as UTC.
func localMidnight(c time.Time, loc *time.Location) time.Time {
	return time.Date(c.Year(), c.Month(), c.Day(), 0, 0, 0, 0, loc).UTC()
}

func days(a, b time.Time) int { return int(b.Sub(a).Hours() / 24) } // civil days only

func plural(n int, one, many string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s", one)
	}
	return fmt.Sprintf("%d %s", n, many)
}

// item makes one queue entry. today is the civil day of "now"; at is the civil
// day it is shown on (week plan), so wording matches the day it sits on.
func item(c domain.CourseState, k domain.ConceptState, today, at time.Time, loc *time.Location) domain.TodayItem {
	it := domain.TodayItem{ConceptID: k.ConceptID, ConceptName: k.Name, SubjectID: c.Subject.ID,
		Kind: domain.TodayRecall, Minutes: recallMinutes}
	if !k.Taught {
		it.Kind, it.Minutes = domain.TodayTeach, teachMinutes
	}
	if k.DueAt == nil {
		it.DueAt = localMidnight(today, loc)
		it.Reason = "Not taught yet"
		return it
	}
	it.DueAt = *k.DueAt
	switch due := civil(*k.DueAt, loc); {
	case due.Before(today):
		it.Reason = "Overdue by " + plural(days(due, today), "day", "days")
	case due.Equal(today):
		it.Reason = "Due today"
	default:
		it.Reason = "Due " + due.Format("Mon, Jan 2")
	}
	return it
}

func sortItems(its []domain.TodayItem) {
	sort.SliceStable(its, func(i, j int) bool {
		a, b := its[i], its[j]
		if !a.DueAt.Equal(b.DueAt) {
			return a.DueAt.Before(b.DueAt)
		}
		if a.ConceptName != b.ConceptName {
			return a.ConceptName < b.ConceptName
		}
		return a.ConceptID < b.ConceptID
	})
}

// usedMinutes counts the week's events: a teach-back is 15 min, a card review 5.
func usedMinutes(evs []domain.ReviewEvent, from, to time.Time) int {
	n := 0
	for _, e := range evs {
		if e.CreatedAt.Before(from) || !e.CreatedAt.Before(to) {
			continue
		}
		switch e.EventType {
		case domain.EventTeachBack:
			n += teachMinutes
		case domain.EventCardReview:
			n += recallMinutes
		}
	}
	return n
}

// courseHours builds "used / budget". The budget only rises, to cover the work due
// this week, in 0.5 h steps, and the reason says why.
func courseHours(c domain.CourseState, used, dueMinutes, dueCount int) domain.CourseHours {
	h := domain.CourseHours{SubjectID: c.Subject.ID, Name: c.Subject.Name,
		HoursUsed: float64(used) / 60, HoursBudget: c.Subject.WeeklyHours}
	need := float64(used+dueMinutes) / 60
	if need > h.HoursBudget {
		up := math.Ceil((need-h.HoursBudget)*2) / 2
		h.HoursBudget += up
		h.Reason = fmt.Sprintf("Up %.1f h: %s due this week", up, plural(dueCount, "topic", "topics"))
	}
	return h
}

func (Today) Today(st domain.StudyState, now time.Time) (domain.Today, error) {
	loc, err := userLoc(st.User)
	if err != nil {
		return domain.Today{}, err
	}
	today := civil(now, loc)
	wk := today.AddDate(0, 0, -((int(today.Weekday()) + 6) % 7))
	from, to := localMidnight(wk, loc), localMidnight(wk.AddDate(0, 0, 7), loc)
	end := localMidnight(today.AddDate(0, 0, 1), loc)
	out := domain.Today{Date: localMidnight(today, loc), Courses: []domain.TodayCourse{}, StudyTime: []domain.CourseHours{}}
	for _, c := range st.Courses {
		tc := domain.TodayCourse{SubjectID: c.Subject.ID, Name: c.Subject.Name, Items: []domain.TodayItem{}}
		wkMin, wkN := 0, 0
		for _, k := range c.Concepts {
			if k.DueAt == nil || k.DueAt.Before(end) {
				it := item(c, k, today, today, loc)
				tc.Items = append(tc.Items, it)
				tc.Minutes += it.Minutes
			}
			if k.DueAt == nil || k.DueAt.Before(to) {
				wkMin += map[bool]int{true: teachMinutes, false: recallMinutes}[!k.Taught]
				wkN++
			}
		}
		sortItems(tc.Items)
		out.Courses = append(out.Courses, tc)
		out.StudyTime = append(out.StudyTime, courseHours(c, usedMinutes(c.Events, from, to), wkMin, wkN))
	}
	return out, nil
}

// Week places each topic on its due day. Overdue and never-taught topics sit on
// today (or on the first day of a future week); topics due later are left out.
func (Today) Week(st domain.StudyState, weekStart, now time.Time) (domain.WeekPlan, error) {
	loc, err := userLoc(st.User)
	if err != nil {
		return domain.WeekPlan{}, err
	}
	today := civil(now, loc)
	wk := civil(weekStart, loc)
	floor := today
	if wk.After(today) {
		floor = wk
	}
	from, to := localMidnight(wk, loc), localMidnight(wk.AddDate(0, 0, 7), loc)
	plan := domain.WeekPlan{WeekStart: from, Courses: []domain.CourseHours{}, Days: []domain.WeekDay{}, Notes: []string{}}
	byDay := make([][]domain.TodayItem, 7)
	overdue, fresh := 0, 0
	for _, c := range st.Courses {
		wkMin, wkN := 0, 0
		for _, k := range c.Concepts {
			day := floor
			if k.DueAt != nil && civil(*k.DueAt, loc).After(floor) {
				day = civil(*k.DueAt, loc)
			}
			i := days(wk, day)
			if i < 0 || i > 6 {
				continue
			}
			it := item(c, k, floor, day, loc)
			byDay[i] = append(byDay[i], it)
			wkMin += it.Minutes
			wkN++
			if k.DueAt == nil {
				fresh++
			} else if civil(*k.DueAt, loc).Before(floor) {
				overdue++
			}
		}
		plan.Courses = append(plan.Courses, courseHours(c, usedMinutes(c.Events, from, to), wkMin, wkN))
	}
	for i := range byDay {
		sortItems(byDay[i])
		if byDay[i] == nil {
			byDay[i] = []domain.TodayItem{}
		}
		plan.Days = append(plan.Days, domain.WeekDay{Date: localMidnight(wk.AddDate(0, 0, i), loc), Items: byDay[i]})
	}
	if overdue > 0 {
		plan.Notes = append(plan.Notes, plural(overdue, "overdue topic is", "overdue topics are")+" moved to "+floor.Format("Mon, Jan 2"))
	}
	if fresh > 0 {
		plan.Notes = append(plan.Notes, plural(fresh, "topic is", "topics are")+" not taught yet")
	}
	return plan, nil
}
