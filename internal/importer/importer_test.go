package importer

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/CrisH2307/repetition-study/internal/domain"
	"github.com/CrisH2307/repetition-study/internal/fakes"
)

const notes = "The mean is the sum divided by n. The median is the middle value when sorted."

func setup(t *testing.T) (*Importer, *fakes.CourseStore, string) {
	t.Helper()
	ctx := context.Background()
	cs := &fakes.CourseStore{}
	c, _ := cs.CreateCourse(ctx, "a", domain.Subject{Name: "Stats", WeeklyHours: 3})
	up, _ := cs.CreateUpload(ctx, "a", domain.Upload{SubjectID: c.ID, Filename: "n"}, nil, notes)
	n := fakes.CourseNotes{Store: cs}
	if err := n.ReadUpload(ctx, "a", up.ID); err != nil {
		t.Fatal(err)
	}
	pages, _ := cs.ListPages(ctx, "a", up.ID)
	if _, err := n.MarkReady(ctx, "a", pages[0].ID); err != nil {
		t.Fatal(err)
	}
	jobs := &fakes.ImportStore{Owners: map[string]string{c.ID: "a"}}
	return &Importer{Jobs: jobs, Courses: cs, Review: &fakes.ReviewStore{}, Notes: n}, cs, c.ID
}

func TestImportRows(t *testing.T) {
	im, cs, sub := setup(t)
	csv := "Concept,Claim,Kind,Quote\n" +
		"Mean,mean is sum over n,definition,the sum divided by n\n" + // ok
		"Mean,second claim,statement,the middle value when sorted\n" + // ok, same concept
		"Mean,bad quote,definition,not in the notes at all\n" + // row 4
		"Mean,bad kind,opinion,the sum divided by n\n" + // row 5
		"=1+1,claim,definition,the sum divided by n\n" + // ok, name is never echoed
		"Mean,short\n" // row 7
	j, err := im.Import(context.Background(), "a", sub, "f.csv", []byte(csv))
	if err != nil {
		t.Fatal(err)
	}
	if j.Status != domain.ImportDone || j.RowsTotal != 6 || j.RowsImported != 3 || len(j.RowErrors) != 3 {
		t.Fatalf("%+v", j)
	}
	for i, want := range []int{4, 5, 7} {
		if j.RowErrors[i].Row != want {
			t.Fatalf("row errors: %+v", j.RowErrors)
		}
	}
	if !strings.Contains(j.RowErrors[0].Error, "not on a Ready page") {
		t.Fatalf("%q", j.RowErrors[0].Error)
	}
	_ = cs
}

func TestBadHeaderAndLimit(t *testing.T) {
	im, _, sub := setup(t)
	j, _ := im.Import(context.Background(), "a", sub, "f.csv", []byte("a,b\n1,2\n"))
	if j.Status != domain.ImportFailed || len(j.RowErrors) != 1 {
		t.Fatalf("%+v", j)
	}
	var b strings.Builder
	b.WriteString("concept,claim,kind,quote\n")
	for i := 0; i < 501; i++ {
		fmt.Fprintf(&b, "C%d,c,definition,the sum divided by n\n", i)
	}
	j, _ = im.Import(context.Background(), "a", sub, "f.csv", []byte(b.String()))
	if j.RowsTotal != 500 || j.RowsImported != 500 || len(j.RowErrors) != 1 || j.RowErrors[0].Row != 502 {
		t.Fatalf("%d %d %+v", j.RowsTotal, j.RowsImported, j.RowErrors)
	}
}

func TestOtherUsersCourse(t *testing.T) {
	im, _, sub := setup(t)
	if _, err := im.Import(context.Background(), "b", sub, "f.csv", []byte("concept,claim,kind,quote\n")); err == nil {
		t.Fatal("must refuse another user's course")
	}
}

func TestSafe(t *testing.T) {
	for _, s := range []string{"=1+1", "+1", "-1", "@x", "\tx"} {
		if safe(s)[0] != '\'' {
			t.Fatalf("%q", s)
		}
	}
	if safe("Row ok") != "Row ok" {
		t.Fatal("changed plain text")
	}
}
