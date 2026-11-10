package fakes

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/CrisH2307/repetition-study/internal/domain"
)

func drain(t *testing.T, ch <-chan domain.Delta) string {
	t.Helper()
	var sb strings.Builder
	for d := range ch {
		if d.Err != nil {
			t.Fatal(d.Err)
		}
		sb.WriteString(d.Text)
	}
	return sb.String()
}

func TestLLMIsDeterministic(t *testing.T) {
	ctx := context.Background()
	run := func() (string, map[string]int) {
		l := &LLM{Reply: "one two three", JSONReply: `{"a":1}`}
		ch, err := l.Stream(ctx, domain.LLMRequest{})
		if err != nil {
			t.Fatal(err)
		}
		var out map[string]int
		if err := l.JSON(ctx, domain.LLMRequest{}, &out); err != nil {
			t.Fatal(err)
		}
		return drain(t, ch), out
	}
	s1, j1 := run()
	s2, j2 := run()
	if s1 != "one two three" || s1 != s2 || !reflect.DeepEqual(j1, j2) || j1["a"] != 1 {
		t.Fatalf("not deterministic: %q %q %v %v", s1, s2, j1, j2)
	}
}

func TestLLMRecordsRequests(t *testing.T) {
	l := &LLM{}
	_, _ = l.Stream(context.Background(), domain.LLMRequest{System: "sys"})
	if len(l.Requests) != 1 || l.Requests[0].System != "sys" {
		t.Fatalf("requests not recorded: %+v", l.Requests)
	}
}

func TestSchedulerTeachBackMovesDueDate(t *testing.T) {
	ctx := context.Background()
	clock := NewClock()
	s := &Scheduler{Clock: clock}
	wrong := []domain.ClaimResult{{Label: domain.LabelContradicted}}
	right := []domain.ClaimResult{{Label: domain.LabelSupported}}

	good, _ := s.ApplyTeachBack(ctx, "c1", right)
	again, _ := s.ApplyTeachBack(ctx, "c1", wrong)
	if *good.Grade != domain.GradeGood || *again.Grade != domain.GradeAgain {
		t.Fatalf("grades: %v %v", *good.Grade, *again.Grade)
	}
	if !again.DueAfter.Before(again.DueBefore) {
		t.Fatalf("a wrong explanation must pull the review earlier: %v -> %v", again.DueBefore, again.DueAfter)
	}
	if len(s.Events) != 2 {
		t.Fatalf("events: %d", len(s.Events))
	}

	clock.Advance(48 * time.Hour)
	due, _ := s.Due(ctx, clock.Now())
	if len(due) != 1 || due[0].ConceptID != "c1" {
		t.Fatalf("due: %+v", due)
	}
}

func TestOtherFakesRun(t *testing.T) {
	ctx := context.Background()
	te := &TeachEngine{}
	sess, _ := te.Start(ctx, "c1")
	ch, err := te.Turn(ctx, sess.ID, "markov chains forget the past", "")
	if err != nil || drain(t, ch) == "" {
		t.Fatalf("turn: %v", err)
	}
	if r, err := te.Finish(ctx, sess.ID); err != nil || r.ConceptID != "c1" {
		t.Fatalf("finish: %+v %v", r, err)
	}
	if _, err := te.Turn(ctx, "nope", "", ""); err == nil {
		t.Fatal("unknown session must error")
	}

	res, _ := (&Checker{}).Check(ctx, domain.CheckInput{Concept: domain.Concept{Claims: []domain.Claim{
		{ID: "a", Required: true}, {ID: "b"},
	}}})
	if len(res) != 1 || *res[0].ClaimID != "a" {
		t.Fatalf("checker: %+v", res)
	}
	p, _ := Planner{}.Generate(ctx, domain.DemoUserID, Epoch)
	d, _ := Digester{}.Today(ctx, Epoch)
	if len(p.Blocks) != 1 || len(d.Items) != 1 {
		t.Fatalf("planner/digester: %+v %+v", p, d)
	}
}
