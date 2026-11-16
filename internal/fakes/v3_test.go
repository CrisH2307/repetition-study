package fakes

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/CrisH2307/repetition-study/internal/domain"
)

func TestAuthFlow(t *testing.T) {
	ctx := context.Background()
	clock := NewClock()
	a := &Auth{Clock: clock}
	inv, _ := a.NewInvite(ctx, "Dino")
	u1, s1, _, err := a.Redeem(ctx, inv)
	if err != nil || u1.Name != "Dino" {
		t.Fatalf("redeem: %+v %v", u1, err)
	}
	u2, _, _, _ := a.Redeem(ctx, inv)
	if u2.ID != u1.ID {
		t.Fatal("a reused invite must sign in the same user")
	}
	if got, err := a.UserFor(ctx, s1); err != nil || got.ID != u1.ID {
		t.Fatalf("session: %+v %v", got, err)
	}
	if _, err := a.UserFor(ctx, s1+"x"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatal("tampered session accepted")
	}
	clock.Advance(31 * 24 * time.Hour)
	if _, err := a.UserFor(ctx, s1); !errors.Is(err, domain.ErrNotFound) {
		t.Fatal("expired session accepted")
	}
	_, s3, _, _ := a.Redeem(ctx, inv)
	_ = a.SignOut(ctx, s3)
	if _, err := a.UserFor(ctx, s3); !errors.Is(err, domain.ErrNotFound) {
		t.Fatal("signed-out session accepted")
	}
}

func TestCourseLoopAndIsolation(t *testing.T) {
	ctx := context.Background()
	st := &CourseStore{}
	n := CourseNotes{Store: st}
	c, _ := st.CreateCourse(ctx, "a", domain.Subject{Name: "Stats", WeeklyHours: 3})
	k, _ := st.CreateConcept(ctx, "a", c.ID, "Mean")
	up, err := st.CreateUpload(ctx, "a", domain.Upload{SubjectID: c.ID, Filename: "notes"}, nil, "The mean is the sum divided by n.")
	if err != nil {
		t.Fatal(err)
	}
	if err := n.ReadUpload(ctx, "a", up.ID); err != nil {
		t.Fatal(err)
	}
	pages, _ := st.ListPages(ctx, "a", up.ID)
	if len(pages) != 1 {
		t.Fatalf("pages: %+v", pages)
	}
	in := domain.ClaimInput{Text: "mean = sum / n", Kind: domain.KindDefinition, Required: true,
		Anchor: domain.Anchor{Quote: "the sum divided by n"}}
	var ue *domain.UserError
	if _, err := n.AddClaim(ctx, "a", k.ID, in); !errors.As(err, &ue) {
		t.Fatalf("a quote from a non-Ready page must be refused, got %v", err)
	}
	if _, err := n.MarkReady(ctx, "a", pages[0].ID); err != nil {
		t.Fatal(err)
	}
	blocks, _ := st.ReadyBlocks(ctx, "a", c.ID)
	in.Anchor.BlockID = blocks[0].ID
	cl, err := n.AddClaim(ctx, "a", k.ID, in)
	if err != nil || cl.Anchor == nil || *cl.Kind != domain.KindDefinition {
		t.Fatalf("claim: %+v %v", cl, err)
	}
	if _, err := n.MarkNotReady(ctx, "a", pages[0].ID); !errors.As(err, &ue) {
		t.Fatal("un-ready must be refused while a claim quotes the page")
	}
	src, _ := st.CourseSources(ctx, "a", c.ID)
	if len(src) != 1 || src[0].ID != cl.SourceID {
		t.Fatalf("sources: %+v", src)
	}

	// User b sees none of a's rows.
	for name, err := range map[string]error{
		"course":  func() error { _, e := st.GetCourse(ctx, "b", c.ID); return e }(),
		"concept": func() error { _, e := st.GetConcept(ctx, "b", k.ID); return e }(),
		"upload":  func() error { _, e := st.GetUpload(ctx, "b", up.ID); return e }(),
		"page":    func() error { _, e := st.GetPage(ctx, "b", pages[0].ID); return e }(),
		"claim":   func() error { _, e := n.EditClaim(ctx, "b", cl.ID, in); return e }(),
		"delete":  st.DeleteClaim(ctx, "b", cl.ID),
	} {
		if !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("%s: user b got %v, want ErrNotFound", name, err)
		}
	}
	if l, _ := st.ListCourses(ctx, "b"); len(l) != 0 {
		t.Errorf("user b lists %d courses", len(l))
	}
}

func TestReviewStoreScopesSessions(t *testing.T) {
	ctx := context.Background()
	r := &ReviewStore{Sessions: map[string]string{"s1": "a"}}
	if err := r.SaveSelfGrades(ctx, "b", "s1", nil); !errors.Is(err, domain.ErrNotFound) {
		t.Fatal("user b self-graded a's session")
	}
	d, err := r.CreateDispute(ctx, "a", domain.Dispute{SessionID: "s1", Note: "quote is fine"})
	if err != nil || d.Status != domain.DisputeOpen {
		t.Fatalf("dispute: %+v %v", d, err)
	}
	if all, _ := r.AllDisputes(ctx); len(all) != 1 {
		t.Fatalf("disputes: %+v", all)
	}
}

func TestTodayPlanner(t *testing.T) {
	past := Epoch.AddDate(0, 0, -1)
	later := Epoch.AddDate(0, 0, 5)
	st := domain.StudyState{Courses: []domain.CourseState{{
		Subject: domain.Subject{ID: "c1", Name: "Stats", WeeklyHours: 3},
		Concepts: []domain.ConceptState{
			{ConceptID: "new", Name: "New"},
			{ConceptID: "due", Name: "Due", DueAt: &past, Taught: true},
			{ConceptID: "later", Name: "Later", DueAt: &later, Taught: true},
		},
	}}}
	today, _ := TodayPlanner{}.Today(st, Epoch)
	c := today.Courses[0]
	if len(c.Items) != 2 || c.Minutes != 20 || c.Items[0].Kind != domain.TodayTeach {
		t.Fatalf("today: %+v", c)
	}
	w, _ := TodayPlanner{}.Week(st, Epoch.Add(-9*time.Hour), Epoch)
	if len(w.Days) != 7 || len(w.Days[0].Items) != 2 {
		t.Fatalf("week: %+v", w)
	}
}
