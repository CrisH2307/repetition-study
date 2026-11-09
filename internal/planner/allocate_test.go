package planner

import (
	"reflect"
	"testing"
	"time"

	"github.com/CrisH2307/repetition-study/internal/domain"
)

var (
	week = time.Date(2026, 11, 16, 0, 0, 0, 0, time.UTC) // Monday
	now  = time.Date(2026, 11, 9, 12, 0, 0, 0, time.UTC)
	exam = time.Date(2026, 12, 15, 0, 0, 0, 0, time.UTC)
)

// subject builds a subject with n concepts, all reviewed 5 days before now with the given stability.
func subject(id string, hours float64, lvl domain.Level, n int, stability float64) SubjectData {
	e := exam
	sd := SubjectData{Subject: domain.Subject{ID: id, Name: "Subject " + id, WeeklyHours: hours, Level: lvl, ExamDate: &e, Weight: 1}}
	for i := 0; i < n; i++ {
		sd.Concepts = append(sd.Concepts, ConceptData{
			ID: id + "-c" + string(rune('a'+i)), Name: "Concept " + string(rune('a'+i)),
			Stability: stability, LastReview: now.AddDate(0, 0, -5), Scheduled: true,
		})
	}
	return sd
}

func count(blocks []domain.PlanBlock, subjectID string) int {
	n := 0
	for _, b := range blocks {
		if b.SubjectID == subjectID {
			n++
		}
	}
	return n
}

func TestDeterministic(t *testing.T) {
	in := []SubjectData{subject("b", 4, domain.LevelBeginner, 3, 4), subject("a", 6, domain.LevelAdvanced, 4, 9)}
	a1, err := DefaultConfig().Allocate(now, week, in, nil)
	if err != nil {
		t.Fatal(err)
	}
	a2, _ := DefaultConfig().Allocate(now, week, []SubjectData{in[1], in[0]}, nil)
	if !reflect.DeepEqual(a1, a2) {
		t.Fatal("same input produced different plans")
	}
}

func TestWeeklyHoursHoldWithLockedBlocks(t *testing.T) {
	in := []SubjectData{subject("a", 3, domain.LevelBeginner, 4, 1), subject("b", 2, domain.LevelBeginner, 4, 1)}
	// Two locked blocks for a, in the first free slots.
	locked := []domain.PlanBlock{
		{ID: "l1", SubjectID: "a", Kind: domain.PlanReview, StartsAt: week.Add(9 * time.Hour), Minutes: 30, Locked: true},
		{ID: "l2", SubjectID: "a", Kind: domain.PlanReview, StartsAt: week.Add(9*time.Hour + 30*time.Minute), Minutes: 30, Locked: true},
	}
	al, err := DefaultConfig().Allocate(now, week, in, locked)
	if err != nil {
		t.Fatal(err)
	}
	if got := count(al.Blocks, "a") + 2; got > 6 {
		t.Fatalf("subject a has %d blocks, weekly cap is 6", got)
	}
	if got := count(al.Blocks, "b"); got > 4 {
		t.Fatalf("subject b has %d blocks, weekly cap is 4", got)
	}
	for _, b := range al.Blocks {
		for _, l := range locked {
			if b.StartsAt.Before(l.StartsAt.Add(30*time.Minute)) && l.StartsAt.Before(b.StartsAt.Add(30*time.Minute)) {
				t.Fatalf("block at %s overlaps locked block %s", b.StartsAt, l.ID)
			}
		}
	}
}

func TestWorseRetentionRaisesHours(t *testing.T) {
	strong := []SubjectData{subject("a", 6, domain.LevelIntermediate, 3, 40), subject("b", 6, domain.LevelIntermediate, 3, 40)}
	weak := []SubjectData{subject("a", 6, domain.LevelIntermediate, 3, 2), subject("b", 6, domain.LevelIntermediate, 3, 40)}
	s, _ := DefaultConfig().Allocate(now, week, strong, nil)
	w, _ := DefaultConfig().Allocate(now, week, weak, nil)
	if count(w.Blocks, "a") <= count(s.Blocks, "a") {
		t.Fatalf("weaker subject a got %d blocks, stronger got %d", count(w.Blocks, "a"), count(s.Blocks, "a"))
	}
}

func TestWorseRetentionTakesBlocksFromOthersUnderBudget(t *testing.T) {
	cfg := DefaultConfig()
	cfg.MaxBlocks = 8
	weak := []SubjectData{subject("a", 6, domain.LevelIntermediate, 3, 1), subject("b", 6, domain.LevelIntermediate, 3, 30)}
	al, _ := cfg.Allocate(now, week, weak, nil)
	if count(al.Blocks, "a") <= count(al.Blocks, "b") {
		t.Fatalf("a=%d b=%d, the weaker subject should win the budget", count(al.Blocks, "a"), count(al.Blocks, "b"))
	}
}

func TestTeachBeforeReviewAndLeadTime(t *testing.T) {
	cfg := DefaultConfig()
	// Plan the current week: now is Tuesday noon, so nothing starts before now and teach needs 24h.
	cur := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	start := time.Date(2026, 11, 9, 0, 0, 0, 0, time.UTC)
	in := []SubjectData{subject("a", 6, domain.LevelBeginner, 3, 2)}
	al, err := cfg.Allocate(cur, start, in, nil)
	if err != nil {
		t.Fatal(err)
	}
	teach := map[string]time.Time{}
	for _, b := range al.Blocks {
		if b.StartsAt.Before(cur) {
			t.Fatalf("block in the past: %s", b.StartsAt)
		}
		if b.Reason == "" {
			t.Fatal("block without reason")
		}
		if b.Kind == domain.PlanTeach {
			if b.StartsAt.Before(cur.Add(24 * time.Hour)) {
				t.Fatalf("teach at %s is under a day ahead", b.StartsAt)
			}
			teach[*b.ConceptID] = b.StartsAt
		}
	}
	reviews := 0
	for _, b := range al.Blocks {
		if b.Kind == domain.PlanReview {
			reviews++
			ts, ok := teach[*b.ConceptID]
			if !ok || !ts.Before(b.StartsAt) {
				t.Fatalf("review of %s has no earlier teach block", *b.ConceptID)
			}
		}
	}
	if reviews == 0 || len(teach) == 0 {
		t.Fatalf("expected teach and review blocks, got %d teach %d review", len(teach), reviews)
	}
}

func TestSubjectWithNoConceptsUsesWeeklyHoursAndLevel(t *testing.T) {
	e := exam
	empty := SubjectData{Subject: domain.Subject{ID: "z", Name: "Empty", WeeklyHours: 3, Level: domain.LevelBeginner, ExamDate: &e, Weight: 1}}
	al, err := DefaultConfig().Allocate(now, week, []SubjectData{empty}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if n := count(al.Blocks, "z"); n != 6 {
		t.Fatalf("want 6 blocks from 3 weekly hours, got %d", n)
	}
	for _, b := range al.Blocks {
		if b.Kind != domain.PlanNew || b.ConceptID != nil {
			t.Fatalf("no-concept subject should get new blocks without a concept, got %+v", b)
		}
	}
}

func TestNewShareReserved(t *testing.T) {
	in := []SubjectData{subject("a", 5, domain.LevelBeginner, 4, 1), subject("b", 5, domain.LevelBeginner, 4, 1)}
	al, _ := DefaultConfig().Allocate(now, week, in, nil)
	n := 0
	for _, b := range al.Blocks {
		if b.Kind == domain.PlanNew {
			n++
		}
	}
	if want := 4; n != want { // 30% of 14 blocks, rounded
		t.Fatalf("want %d new blocks of %d, got %d", want, len(al.Blocks), n)
	}
}

func TestLevelFadesWithEvidence(t *testing.T) {
	cfg := DefaultConfig()
	s := subject("a", 4, domain.LevelBeginner, 2, 3)
	fresh := cfg.Score(s, now, week)
	s.Events = 20
	faded := cfg.Score(s, now, week)
	if fresh.LevelNow != 1.3 || faded.LevelNow != 1 {
		t.Fatalf("level_now fresh=%v faded=%v, want 1.3 and 1", fresh.LevelNow, faded.LevelNow)
	}
}
