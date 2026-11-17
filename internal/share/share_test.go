package share_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/CrisH2307/repetition-study/internal/course"
	"github.com/CrisH2307/repetition-study/internal/domain"
	"github.com/CrisH2307/repetition-study/internal/fakes"
	"github.com/CrisH2307/repetition-study/internal/share"
)

var bg = context.Background()

type users map[string]domain.User

func (u users) UserByID(_ context.Context, id string) (domain.User, error) {
	x, ok := u[id]
	if !ok {
		return x, domain.ErrNotFound
	}
	return x, nil
}

type world struct {
	t     *testing.T
	cs    *fakes.CourseStore
	notes course.Notes
	rev   *fakes.ReviewStore
	svc   *share.Service
	us    users
	cards map[string]map[string][]domain.StudyCard // user -> course -> cards
}

func newWorld(t *testing.T) *world {
	cs := &fakes.CourseStore{}
	w := &world{t: t, cs: cs, notes: course.Notes{Store: cs}, us: users{
		"teacher": {ID: "teacher", Name: "Ms Rao", IsInstructor: true},
		"ann":     {ID: "ann", Name: "Ann"},
		"bob":     {ID: "bob", Name: "Bob"},
	}, cards: map[string]map[string][]domain.StudyCard{}}
	w.rev = &fakes.ReviewStore{Views: map[string]map[string]domain.CourseView{}}
	w.svc = &share.Service{Store: &fakes.ShareStore{}, Users: w.us, Courses: cs, Review: w.rev, Notes: w.notes}
	return w
}

func ok(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

// course makes a course for user with one Ready note.
func (w *world) course(user, name, noteText string) domain.Subject {
	c, err := w.cs.CreateCourse(bg, user, domain.Subject{Name: name, WeeklyHours: 1, Level: domain.Level("intermediate"), Weight: 1})
	ok(w.t, err)
	if noteText != "" {
		u, err := w.cs.CreateUpload(bg, user, domain.Upload{SubjectID: c.ID, Filename: user + "-notes.txt"}, nil, noteText)
		ok(w.t, err)
		ok(w.t, w.notes.ReadUpload(bg, user, u.ID))
		pgs, err := w.cs.ListPages(bg, user, u.ID)
		ok(w.t, err)
		for _, p := range pgs {
			_, err := w.notes.MarkReady(bg, user, p.ID)
			ok(w.t, err)
		}
	}
	w.sync(user, c.ID)
	return c
}

func (w *world) concept(user, courseID, name string) domain.Concept {
	c, err := w.cs.CreateConcept(bg, user, courseID, name)
	ok(w.t, err)
	w.sync(user, courseID, c.ID)
	return c
}

// sync rebuilds the fake course view (concept ids are appended).
func (w *world) sync(user, courseID string, add ...string) {
	if w.rev.Views[user] == nil {
		w.rev.Views[user] = map[string]domain.CourseView{}
	}
	v := w.rev.Views[user][courseID]
	for _, id := range add {
		c, _ := w.cs.GetConcept(bg, user, id)
		v.Cards = append(v.Cards, domain.StudyCard{ConceptID: id, Name: c.Name})
	}
	w.rev.Views[user][courseID] = v
}

func (w *world) claim(user, conceptID, text, quote string) {
	blocks, err := w.cs.ReadyBlocks(bg, user, mustSubject(w, user, conceptID))
	ok(w.t, err)
	_, err = w.notes.AddClaim(bg, user, conceptID, domain.ClaimInput{Text: text, Kind: domain.KindDefinition, Required: true,
		Anchor: domain.Anchor{BlockID: blocks[0].ID, Quote: quote}})
	ok(w.t, err)
}

func mustSubject(w *world, user, conceptID string) string {
	c, err := w.cs.GetConcept(bg, user, conceptID)
	ok(w.t, err)
	return c.SubjectID
}

const (
	quote = "Entropy is the average uncertainty of a random variable."
	text  = "Entropy is the average uncertainty in a random variable"
)

func ownerPack(w *world, owner string) (domain.Subject, domain.Concept) {
	c := w.course(owner, "Information theory", quote+" Mutual information is symmetric.")
	k := w.concept(owner, c.ID, "Entropy")
	w.claim(owner, k.ID, text, quote)
	return c, k
}

func TestPackCarriesNoQuoteOrNotes(t *testing.T) {
	w := newWorld(t)
	c, _ := ownerPack(w, "ann")
	n, err := w.svc.Create(bg, "ann", c.ID, false)
	ok(t, err)
	p, err := w.svc.Preview(bg, n.Token)
	ok(t, err)
	b, _ := json.Marshal(p)
	if strings.Contains(string(b), quote) || strings.Contains(string(b), "Mutual information") {
		t.Fatalf("preview leaks course text: %s", b)
	}
	if p.Instructor || len(p.Sources) != 0 || p.OwnerName != "Ann" || p.CourseName != "Information theory" ||
		len(p.Concepts) != 1 || p.Concepts[0].Claims[0].Text != text {
		t.Fatalf("preview: %+v", p)
	}
}

func TestImportAnchorsToReceiversOwnNotes(t *testing.T) {
	w := newWorld(t)
	c, _ := ownerPack(w, "ann")
	n, _ := w.svc.Create(bg, "ann", c.ID, false)

	mine := w.course("bob", "Info", "In class: "+quote+" Next we saw channels.")
	got, err := w.svc.Import(bg, "bob", n.Token, mine.ID)
	ok(t, err)
	if got.ConceptsAdded != 1 || got.ClaimsAdded != 1 || len(got.NeedsQuote) != 0 || got.UploadsCopied != 0 {
		t.Fatalf("import: %+v", got)
	}
	// the claim is anchored in bob's block, not ann's
	blocks, _ := w.cs.ReadyBlocks(bg, "bob", mine.ID)
	cl, _ := w.cs.ListClaims(bg, "bob", w.lastConcept("bob"))
	if len(cl) != 1 || cl[0].Anchor == nil || cl[0].Anchor.BlockID != blocks[0].ID {
		t.Fatalf("claims: %+v", cl)
	}
}

func (w *world) lastConcept(user string) string {
	for i := 1; i < 20; i++ {
		id := "concept-" + string(rune('0'+i))
		if c, err := w.cs.GetConcept(bg, user, id); err == nil && c.Name == "Entropy" {
			return id
		}
	}
	w.t.Fatal("no Entropy concept for " + user)
	return ""
}

func TestImportWithoutNotesListsNeedsQuote(t *testing.T) {
	w := newWorld(t)
	c, _ := ownerPack(w, "ann")
	n, _ := w.svc.Create(bg, "ann", c.ID, false)

	empty := w.course("bob", "Info", "Totally unrelated text about cooking pasta in salted water.")
	got, err := w.svc.Import(bg, "bob", n.Token, empty.ID)
	ok(t, err)
	if got.ClaimsAdded != 0 || len(got.NeedsQuote) != 1 || got.NeedsQuote[0].Claim.Text != text {
		t.Fatalf("import: %+v", got)
	}
	// non-instructor pack never copies notes, even though ann has Ready notes
	if ups, _ := w.cs.ListUploads(bg, "bob", empty.ID); len(ups) != 1 || got.UploadsCopied != 0 {
		t.Fatalf("uploads: %+v", ups)
	}
}

func TestInstructorPackCopiesNotes(t *testing.T) {
	w := newWorld(t)
	c, _ := ownerPack(w, "teacher")
	// a non-instructor cannot ask for notes
	_, err := w.svc.Create(bg, "ann", w.course("ann", "x", "").ID, true)
	var ue *domain.UserError
	if !errors.As(err, &ue) {
		t.Fatalf("want UserError, got %v", err)
	}
	n, err := w.svc.Create(bg, "teacher", c.ID, true)
	ok(t, err)
	p, _ := w.svc.Preview(bg, n.Token)
	if !p.Instructor || len(p.Sources) != 1 || p.Sources[0] != "teacher-notes.txt" {
		t.Fatalf("preview: %+v", p)
	}
	mine := w.course("bob", "Info", "")
	got, err := w.svc.Import(bg, "bob", n.Token, mine.ID)
	ok(t, err)
	if got.UploadsCopied != 1 || got.ClaimsAdded != 1 || len(got.NeedsQuote) != 0 {
		t.Fatalf("import: %+v", got)
	}
	if bl, _ := w.cs.ReadyBlocks(bg, "bob", mine.ID); len(bl) == 0 {
		t.Fatal("copied notes are not Ready")
	}
}

func TestRevokedAndForeignAccess(t *testing.T) {
	w := newWorld(t)
	c, _ := ownerPack(w, "ann")
	n, _ := w.svc.Create(bg, "ann", c.ID, false)
	mine := w.course("bob", "Info", quote)

	// another user's course and share are 404
	if _, err := w.svc.Create(bg, "bob", c.ID, false); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("create on foreign course: %v", err)
	}
	if err := w.svc.Revoke(bg, "bob", n.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("revoke foreign: %v", err)
	}
	if _, err := w.svc.Import(bg, "bob", n.Token, c.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("import into ann's course: %v", err)
	}
	if _, err := w.svc.Preview(bg, "not-a-token"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("unknown token: %v", err)
	}
	ok(t, w.svc.Revoke(bg, "ann", n.ID))
	if _, err := w.svc.Preview(bg, n.Token); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("revoked preview: %v", err)
	}
	if _, err := w.svc.Import(bg, "bob", n.Token, mine.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("revoked import: %v", err)
	}
}
