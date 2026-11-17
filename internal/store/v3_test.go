package store_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/CrisH2307/repetition-study/internal/domain"
	"github.com/CrisH2307/repetition-study/internal/store"
)

var bg = context.Background()

func mustOK(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func isNF(t *testing.T, what string, err error) {
	t.Helper()
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("%s: want ErrNotFound, got %v", what, err)
	}
}

func isUser(t *testing.T, what string, err error) {
	t.Helper()
	var ue *domain.UserError
	if !errors.As(err, &ue) {
		t.Errorf("%s: want UserError, got %v", what, err)
	}
}

func newUser(t *testing.T, s *store.Store, name string) domain.User {
	t.Helper()
	h := []byte("invite-" + name)
	_, err := s.CreateInvite(bg, name, h)
	mustOK(t, err)
	u, err := s.RedeemInvite(bg, h, time.Now())
	mustOK(t, err)
	return u
}

func TestAuthStore(t *testing.T) {
	connect := openDB(t)
	s := store.New(connect())
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)

	inv, err := s.CreateInvite(bg, "Dino", []byte("h1"))
	mustOK(t, err)
	if inv.ID == "" || inv.UsedAt != nil || inv.UserID != nil {
		t.Fatalf("invite: %+v", inv)
	}
	u1, err := s.RedeemInvite(bg, []byte("h1"), now)
	mustOK(t, err)
	u2, err := s.RedeemInvite(bg, []byte("h1"), now.Add(time.Hour))
	mustOK(t, err)
	if u1.ID == "" || u1 != u2 || u1.Name != "Dino" || u1.TimeZone == "" {
		t.Fatalf("redeem twice: %+v vs %+v", u1, u2)
	}
	_, err = s.RedeemInvite(bg, []byte("nope"), now)
	isNF(t, "unknown invite", err)

	mustOK(t, s.CreateSession(bg, u1.ID, []byte("s1"), now.Add(time.Hour)))
	got, err := s.SessionUser(bg, []byte("s1"), now)
	if err != nil || got != u1 {
		t.Fatalf("SessionUser: %+v %v", got, err)
	}
	_, err = s.SessionUser(bg, []byte("s1"), now.Add(time.Hour))
	isNF(t, "expired session", err)
	_, err = s.SessionUser(bg, []byte("zzz"), now)
	isNF(t, "unknown session", err)
	mustOK(t, s.DeleteSession(bg, []byte("s1")))
	_, err = s.SessionUser(bg, []byte("s1"), now)
	isNF(t, "deleted session", err)
	mustOK(t, s.DeleteSession(bg, []byte("s1"))) // twice is fine
}

func setup(t *testing.T) (*store.Store, domain.User, domain.User, domain.Subject, domain.Concept, domain.Upload, domain.Page) {
	t.Helper()
	connect := openDB(t)
	s := store.New(connect())
	a, b := newUser(t, s, "A"), newUser(t, s, "B")
	c, err := s.CreateCourse(bg, a.ID, domain.Subject{Name: "Calc", WeeklyHours: 3, Level: domain.LevelBeginner, Weight: 1})
	mustOK(t, err)
	k, err := s.CreateConcept(bg, a.ID, c.ID, "Limits")
	mustOK(t, err)
	up, err := s.CreateUpload(bg, a.ID, domain.Upload{SubjectID: c.ID, Filename: "notes.txt"}, nil, "A limit is a value.")
	mustOK(t, err)
	mustOK(t, s.SavePages(bg, a.ID, up.ID, []domain.Page{{Number: 1, RawText: "A limit is a value.", EditedText: "A limit is a value."}}, [][]byte{[]byte("png")}))
	pages, err := s.ListPages(bg, a.ID, up.ID)
	mustOK(t, err)
	if len(pages) != 1 {
		t.Fatalf("pages: %+v", pages)
	}
	return s, a, b, c, k, up, pages[0]
}

func readyBlock(text string) domain.SourceBlock {
	return domain.SourceBlock{Kind: domain.BlockText, TextMD: text, TextNorm: strings.ToLower(text)}
}

func TestCoursePagesAndClaims(t *testing.T) {
	s, a, _, c, k, up, pg := setup(t)

	if up.Status != domain.UploadReading || up.PagesTotal != 0 {
		t.Fatalf("new upload: %+v", up)
	}
	got, _ := s.GetUpload(bg, a.ID, up.ID)
	if got.PagesTotal != 1 || got.PagesReady != 0 {
		t.Fatalf("counts: %+v", got)
	}
	if !pg.HasImage || pg.Status != domain.PageNeedsChecking {
		t.Fatalf("page: %+v", pg)
	}
	img, err := s.PageImage(bg, a.ID, pg.ID)
	if err != nil || string(img) != "png" {
		t.Fatalf("image: %q %v", img, err)
	}
	pdf, pasted, err := s.UploadFile(bg, a.ID, up.ID)
	if err != nil || len(pdf) != 0 || pasted != "A limit is a value." {
		t.Fatalf("file: %q %q %v", pdf, pasted, err)
	}
	mustOK(t, s.SetUploadStatus(bg, a.ID, up.ID, domain.UploadNeedsChecking, ""))

	// Nothing is quotable before Ready.
	blocks, err := s.ReadyBlocks(bg, a.ID, c.ID)
	if err != nil || blocks == nil || len(blocks) != 0 {
		t.Fatalf("ReadyBlocks before: %v %v", blocks, err)
	}
	srcs, _ := s.CourseSources(bg, a.ID, c.ID)
	if len(srcs) != 0 {
		t.Fatalf("sources before: %v", srcs)
	}

	pg, err = s.EditPage(bg, a.ID, pg.ID, "A limit is a number L.")
	mustOK(t, err)
	pg, err = s.SetPageReady(bg, a.ID, pg.ID, []domain.SourceBlock{readyBlock("A limit is a number L."), readyBlock("Second block.")})
	mustOK(t, err)
	if pg.Status != domain.PageReady {
		t.Fatalf("ready: %+v", pg)
	}
	up, _ = s.GetUpload(bg, a.ID, up.ID)
	if up.Status != domain.UploadReady || up.PagesReady != 1 || up.SourceID == nil {
		t.Fatalf("upload after ready: %+v", up)
	}
	blocks, _ = s.ReadyBlocks(bg, a.ID, c.ID)
	if len(blocks) != 2 || blocks[0].TextMD != "A limit is a number L." || blocks[0].SourceID != *up.SourceID {
		t.Fatalf("blocks: %+v", blocks)
	}
	srcs, _ = s.CourseSources(bg, a.ID, c.ID)
	if len(srcs) != 1 || srcs[0].Text != "A limit is a number L.\n\nSecond block." {
		t.Fatalf("sources: %+v", srcs)
	}
	// Ready twice replaces the blocks, not adds.
	_, err = s.SetPageReady(bg, a.ID, pg.ID, []domain.SourceBlock{readyBlock("A limit is a number L.")})
	mustOK(t, err)
	blocks, _ = s.ReadyBlocks(bg, a.ID, c.ID)
	if len(blocks) != 1 {
		t.Fatalf("replace: %+v", blocks)
	}

	// Claims
	in := domain.ClaimInput{Text: "A limit is a number", Kind: domain.KindDefinition, Required: true,
		Anchor: domain.Anchor{BlockID: blocks[0].ID, Quote: "A limit is a number L."}}
	cl, err := s.CreateClaim(bg, a.ID, k.ID, in)
	mustOK(t, err)
	if cl.Kind == nil || *cl.Kind != domain.KindDefinition || cl.Anchor == nil || cl.Anchor.BlockID != blocks[0].ID ||
		cl.AuthorUserID == nil || *cl.AuthorUserID != a.ID || cl.SourceID != *up.SourceID || cl.Locator != "p.1" {
		t.Fatalf("claim: %+v", cl)
	}
	bad := in
	bad.Anchor.BlockID = "00000000-0000-0000-0000-00000000dead"
	_, err = s.CreateClaim(bg, a.ID, k.ID, bad)
	isUser(t, "anchor not on a Ready page", err)
	bad.Anchor.BlockID = "not-a-uuid"
	_, err = s.CreateClaim(bg, a.ID, k.ID, bad)
	isUser(t, "anchor not a uuid", err)
	bad = in
	bad.Kind = "nonsense"
	_, err = s.CreateClaim(bg, a.ID, k.ID, bad)
	isUser(t, "bad kind", err)

	in.Text, in.Required = "A limit is a number L", false
	cl, err = s.UpdateClaim(bg, a.ID, cl.ID, in)
	mustOK(t, err)
	if cl.Text != "A limit is a number L" || cl.Required {
		t.Fatalf("update: %+v", cl)
	}
	con, _ := s.GetConcept(bg, a.ID, k.ID)
	if len(con.Claims) != 1 {
		t.Fatalf("GetConcept claims: %+v", con.Claims)
	}

	// A live claim blocks un-ready, and the page stays Ready.
	_, err = s.SetPageNotReady(bg, a.ID, pg.ID)
	isUser(t, "unready with live claim", err)
	_, err = s.SetPageReady(bg, a.ID, pg.ID, []domain.SourceBlock{readyBlock("x")})
	isUser(t, "re-ready with live claim", err)
	if p, _ := s.GetPage(bg, a.ID, pg.ID); p.Status != domain.PageReady {
		t.Fatalf("page changed: %+v", p)
	}

	// Deleted claims are hidden, the row stays, and un-ready now works.
	mustOK(t, s.DeleteClaim(bg, a.ID, cl.ID))
	isNF(t, "delete twice", s.DeleteClaim(bg, a.ID, cl.ID))
	live, err := s.ListClaims(bg, a.ID, k.ID)
	if err != nil || len(live) != 0 || live == nil {
		t.Fatalf("ListClaims: %v %v", live, err)
	}
	_, err = s.UpdateClaim(bg, a.ID, cl.ID, in)
	isNF(t, "update deleted", err)
	pg, err = s.SetPageNotReady(bg, a.ID, pg.ID)
	mustOK(t, err)
	if pg.Status != domain.PageNeedsChecking {
		t.Fatalf("not ready: %+v", pg)
	}
	blocks, _ = s.ReadyBlocks(bg, a.ID, c.ID)
	srcs, _ = s.CourseSources(bg, a.ID, c.ID)
	up, _ = s.GetUpload(bg, a.ID, up.ID)
	if len(blocks) != 0 || len(srcs) != 0 || up.Status != domain.UploadNeedsChecking {
		t.Fatalf("after unready: %v %v %+v", blocks, srcs, up)
	}

	// Archive hides the course and everything under it.
	mustOK(t, s.ArchiveCourse(bg, a.ID, c.ID))
	_, err = s.GetCourse(bg, a.ID, c.ID)
	isNF(t, "archived course", err)
	_, err = s.GetConcept(bg, a.ID, k.ID)
	isNF(t, "archived concept", err)
	_, err = s.GetUpload(bg, a.ID, up.ID)
	isNF(t, "archived upload", err)
	list, err := s.ListCourses(bg, a.ID)
	if err != nil || len(list) != 0 || list == nil {
		t.Fatalf("ListCourses: %v %v", list, err)
	}
}

func TestCourseValidation(t *testing.T) {
	s, a, _, c, _, _, _ := setup(t)
	_, err := s.CreateCourse(bg, a.ID, domain.Subject{Name: "", Level: domain.LevelBeginner})
	isUser(t, "empty name", err)
	_, err = s.CreateCourse(bg, a.ID, domain.Subject{Name: "x", Level: "expert"})
	isUser(t, "bad level", err)
	_, err = s.CreateConcept(bg, a.ID, c.ID, "Limits")
	isUser(t, "duplicate concept", err)
	_, err = s.CreateUpload(bg, a.ID, domain.Upload{SubjectID: c.ID, Filename: "x"}, nil, "")
	isUser(t, "empty upload", err)
	_, err = s.CreateUpload(bg, a.ID, domain.Upload{SubjectID: c.ID, Filename: "x"}, make([]byte, 20<<20+1), "")
	isUser(t, "big upload", err)
	exam := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)
	c.ExamDate, c.Name = &exam, "Calc II"
	u, err := s.UpdateCourse(bg, a.ID, c)
	if err != nil || u.Name != "Calc II" || u.ExamDate == nil || !u.ExamDate.Equal(exam) {
		t.Fatalf("update: %+v %v", u, err)
	}
	pdf, err := s.CreateUpload(bg, a.ID, domain.Upload{SubjectID: c.ID, Filename: "a.pdf"}, []byte("%PDF"), "")
	mustOK(t, err)
	b, p, err := s.UploadFile(bg, a.ID, pdf.ID)
	if err != nil || string(b) != "%PDF" || p != "" {
		t.Fatalf("pdf file: %q %q %v", b, p, err)
	}
	_, err = s.PageImage(bg, a.ID, "00000000-0000-0000-0000-00000000dead")
	isNF(t, "image of unknown page", err)
}

// The core isolation test: user B can reach nothing of user A.
func TestUserCannotReadAnotherUsersRows(t *testing.T) {
	s, a, b, c, k, up, pg := setup(t)
	pg, err := s.SetPageReady(bg, a.ID, pg.ID, []domain.SourceBlock{readyBlock("A limit is a number L.")})
	mustOK(t, err)
	blocks, _ := s.ReadyBlocks(bg, a.ID, c.ID)
	in := domain.ClaimInput{Text: "t", Kind: domain.KindStatement, Required: true, Anchor: domain.Anchor{BlockID: blocks[0].ID, Quote: "A limit"}}
	cl, err := s.CreateClaim(bg, a.ID, k.ID, in)
	mustOK(t, err)
	sess, err := s.StartSession(bg, k.ID)
	mustOK(t, err)
	d, err := s.CreateDispute(bg, a.ID, domain.Dispute{SessionID: sess.ID, ClaimID: &cl.ID, Note: "wrong"})
	mustOK(t, err)
	if d.UserID != a.ID || d.Status != domain.DisputeOpen || d.CreatedAt.IsZero() {
		t.Fatalf("dispute: %+v", d)
	}

	// Own course for B so a wrong-course id is also tested.
	bc, err := s.CreateCourse(bg, b.ID, domain.Subject{Name: "B", Level: domain.LevelBeginner, Weight: 1})
	mustOK(t, err)
	bk, err := s.CreateConcept(bg, b.ID, bc.ID, "bk")
	mustOK(t, err)

	_, err = s.GetCourse(bg, b.ID, c.ID)
	isNF(t, "GetCourse", err)
	_, err = s.UpdateCourse(bg, b.ID, domain.Subject{ID: c.ID, Name: "hacked", Level: domain.LevelBeginner})
	isNF(t, "UpdateCourse", err)
	isNF(t, "ArchiveCourse", s.ArchiveCourse(bg, b.ID, c.ID))
	_, err = s.CreateConcept(bg, b.ID, c.ID, "x")
	isNF(t, "CreateConcept", err)
	_, err = s.GetConcept(bg, b.ID, k.ID)
	isNF(t, "GetConcept", err)
	_, err = s.CreateUpload(bg, b.ID, domain.Upload{SubjectID: c.ID, Filename: "x"}, nil, "text")
	isNF(t, "CreateUpload", err)
	_, err = s.GetUpload(bg, b.ID, up.ID)
	isNF(t, "GetUpload", err)
	_, err = s.ListUploads(bg, b.ID, c.ID)
	isNF(t, "ListUploads", err)
	_, _, err = s.UploadFile(bg, b.ID, up.ID)
	isNF(t, "UploadFile", err)
	isNF(t, "SetUploadStatus", s.SetUploadStatus(bg, b.ID, up.ID, domain.UploadReady, ""))
	isNF(t, "SavePages", s.SavePages(bg, b.ID, up.ID, []domain.Page{{Number: 2}}, nil))
	_, err = s.ListPages(bg, b.ID, up.ID)
	isNF(t, "ListPages", err)
	_, err = s.GetPage(bg, b.ID, pg.ID)
	isNF(t, "GetPage", err)
	_, err = s.PageImage(bg, b.ID, pg.ID)
	isNF(t, "PageImage", err)
	_, err = s.EditPage(bg, b.ID, pg.ID, "x")
	isNF(t, "EditPage", err)
	_, err = s.SetPageReady(bg, b.ID, pg.ID, nil)
	isNF(t, "SetPageReady", err)
	_, err = s.SetPageNotReady(bg, b.ID, pg.ID)
	isNF(t, "SetPageNotReady", err)
	_, err = s.ReadyBlocks(bg, b.ID, c.ID)
	isNF(t, "ReadyBlocks", err)
	_, err = s.CourseSources(bg, b.ID, c.ID)
	isNF(t, "CourseSources", err)
	_, err = s.ListClaims(bg, b.ID, k.ID)
	isNF(t, "ListClaims", err)
	_, err = s.CreateClaim(bg, b.ID, k.ID, in)
	isNF(t, "CreateClaim into A's concept", err)
	// B's own concept cannot anchor to A's block: not on a Ready page of B's course.
	_, err = s.CreateClaim(bg, b.ID, bk.ID, in)
	isUser(t, "CreateClaim anchored to A's block", err)
	_, err = s.UpdateClaim(bg, b.ID, cl.ID, in)
	isNF(t, "UpdateClaim", err)
	isNF(t, "DeleteClaim", s.DeleteClaim(bg, b.ID, cl.ID))
	isNF(t, "SessionOwner", s.SessionOwner(bg, b.ID, sess.ID))
	isNF(t, "SaveSelfGrades", s.SaveSelfGrades(bg, b.ID, sess.ID, []domain.SelfGrade{{ClaimID: cl.ID, Rating: domain.RatingRight}}))
	_, err = s.CreateDispute(bg, b.ID, domain.Dispute{SessionID: sess.ID, Note: "x"})
	isNF(t, "CreateDispute", err)
	_, err = s.CourseView(bg, b.ID, c.ID)
	isNF(t, "CourseView", err)

	// Nothing leaked into B's lists, and A's rows are untouched.
	list, _ := s.ListCourses(bg, b.ID)
	if len(list) != 1 || list[0].ID != bc.ID {
		t.Fatalf("B's courses: %+v", list)
	}
	st, err := s.StudyState(bg, b.ID, time.Time{})
	if err != nil || len(st.Courses) != 1 || len(st.Courses[0].Concepts) != 1 || st.Courses[0].Concepts[0].ConceptID != bk.ID {
		t.Fatalf("B's study state: %+v %v", st, err)
	}
	got, err := s.GetCourse(bg, a.ID, c.ID)
	if err != nil || got.Name != "Calc" {
		t.Fatalf("A's course changed: %+v %v", got, err)
	}
	mustOK(t, s.SessionOwner(bg, a.ID, sess.ID))
	// Made-up and malformed ids are not found, not a database error.
	_, err = s.GetCourse(bg, a.ID, "nope")
	isNF(t, "bad uuid", err)
}

func TestReviewStore(t *testing.T) {
	connect := openDB(t)
	conn := connect()
	s := store.New(conn)
	a := newUser(t, s, "A")
	c, _ := s.CreateCourse(bg, a.ID, domain.Subject{Name: "C", WeeklyHours: 2, Level: domain.LevelAdvanced, Weight: 1})
	k1, _ := s.CreateConcept(bg, a.ID, c.ID, "k1")
	k2, _ := s.CreateConcept(bg, a.ID, c.ID, "k2")
	up, _ := s.CreateUpload(bg, a.ID, domain.Upload{SubjectID: c.ID, Filename: "n"}, nil, "text")
	mustOK(t, s.SavePages(bg, a.ID, up.ID, []domain.Page{{Number: 7, RawText: "r", EditedText: "e"}}, nil))
	pages, _ := s.ListPages(bg, a.ID, up.ID)
	_, err := s.SetPageReady(bg, a.ID, pages[0].ID, []domain.SourceBlock{readyBlock("The derivative of x squared is 2x.")})
	mustOK(t, err)
	blocks, _ := s.ReadyBlocks(bg, a.ID, c.ID)
	mk := func(text, quote string, req bool) domain.Claim {
		cl, err := s.CreateClaim(bg, a.ID, k1.ID, domain.ClaimInput{Text: text, Kind: domain.KindStatement, Required: req,
			Anchor: domain.Anchor{BlockID: blocks[0].ID, Quote: quote}})
		mustOK(t, err)
		return cl
	}
	c1, c2 := mk("deriv", "derivative of x squared", true), mk("opt", "is 2x", false)

	// Self-grades replace, validate, and never touch events.
	events := func() int { return count(t, conn, `SELECT count(*) FROM review_events`) }
	sess, _ := s.StartSession(bg, k1.ID)
	before := events()
	mustOK(t, s.SaveSelfGrades(bg, a.ID, sess.ID, []domain.SelfGrade{{ClaimID: c1.ID, Rating: domain.RatingWrong}, {ClaimID: c2.ID, Rating: domain.RatingRight}}))
	mustOK(t, s.SaveSelfGrades(bg, a.ID, sess.ID, []domain.SelfGrade{{ClaimID: c1.ID, Rating: domain.RatingNotSure}}))
	if n := count(t, conn, `SELECT count(*) FROM self_grades WHERE session_id = $1 AND rating = 'not_sure'`, sess.ID); n != 1 {
		t.Fatalf("self-grades after replace: %d", n)
	}
	if n := count(t, conn, `SELECT count(*) FROM self_grades`); n != 1 {
		t.Fatalf("self-grades total: %d", n)
	}
	isUser(t, "bad rating", s.SaveSelfGrades(bg, a.ID, sess.ID, []domain.SelfGrade{{ClaimID: c1.ID, Rating: "meh"}}))
	if n := count(t, conn, `SELECT count(*) FROM self_grades`); n != 1 {
		t.Fatalf("failed save must roll back, got %d", n)
	}
	sess2, _ := s.StartSession(bg, k2.ID)
	isUser(t, "claim of another concept", s.SaveSelfGrades(bg, a.ID, sess2.ID, []domain.SelfGrade{{ClaimID: c1.ID, Rating: domain.RatingRight}}))
	_, err = s.CreateDispute(bg, a.ID, domain.Dispute{SessionID: sess2.ID, ClaimID: &c1.ID})
	isUser(t, "dispute claim of another concept", err)
	d1, err := s.CreateDispute(bg, a.ID, domain.Dispute{SessionID: sess.ID, Note: "no claim"})
	mustOK(t, err)
	d2, err := s.CreateDispute(bg, a.ID, domain.Dispute{SessionID: sess.ID, ClaimID: &c1.ID})
	mustOK(t, err)
	all, err := s.AllDisputes(bg)
	if err != nil || len(all) != 2 || all[0].ID != d1.ID || all[1].ID != d2.ID || all[0].ClaimID != nil {
		t.Fatalf("AllDisputes: %+v %v", all, err)
	}
	if events() != before {
		t.Fatal("self-grades and disputes must not write review_events")
	}

	// Labels: one contradicted with a quote on page 7, one supported.
	quote := "derivative of x squared is 2x"
	wrong := "x squared"
	mustOK(t, s.SaveResults(bg, sess.ID, []domain.ClaimResult{
		{ClaimID: &c1.ID, StudentText: "it is 3x", Label: domain.LabelContradicted,
			Citation: &domain.Citation{SourceID: blocks[0].SourceID, Quote: quote, Locator: "p.7"}, Correction: &wrong},
		{ClaimID: &c2.ID, StudentText: "is 2x", Label: domain.LabelSupported},
	}))
	mustOK(t, s.FinishSession(bg, sess.ID))
	if _, err := conn.Exec(bg, `INSERT INTO cards (concept_id, front, back) VALUES ($1, 'f', 'b')`, k1.ID); err != nil {
		t.Fatal(err)
	}
	ins := func(typ, concept string, grade *string, at time.Time) {
		if _, err := conn.Exec(bg, `INSERT INTO review_events (event_type, concept_id, grade, due_before, due_after, stability_before, stability_after, created_at)
			VALUES ($1, $2, $3, now(), now() + interval '1 day', 1, 2, $4)`, typ, concept, grade, at); err != nil {
			t.Fatal(err)
		}
	}
	good, again := "good", "again"
	now := time.Now().UTC().Truncate(time.Microsecond) // Postgres keeps microseconds; Linux clocks have nanoseconds
	ins("card_review", k1.ID, &good, now.Add(-48*time.Hour))
	ins("card_review", k1.ID, &again, now.Add(-24*time.Hour))
	ins("teach_back", k1.ID, nil, now.Add(-time.Hour))
	due := now.Add(72 * time.Hour)
	if _, err := conn.Exec(bg, `INSERT INTO concept_schedules (concept_id, due_at) VALUES ($1, $2)`, k1.ID, due); err != nil {
		t.Fatal(err)
	}

	v, err := s.CourseView(bg, a.ID, c.ID)
	mustOK(t, err)
	if v.Course.ID != c.ID || len(v.Notes) != 1 || len(v.Cards) != 2 || len(v.Mistakes) != 1 {
		t.Fatalf("view: %+v", v)
	}
	c1v := v.Cards[0] // k1 sorts before k2
	if c1v.ConceptID != k1.ID || c1v.Recall == nil || *c1v.Recall != 0.5 || c1v.Explanation == nil || *c1v.Explanation != 0 ||
		c1v.NextDue == nil || !c1v.NextDue.Equal(due) || len(c1v.Claims) != 2 {
		t.Fatalf("card 1: %+v", c1v)
	}
	if k2v := v.Cards[1]; k2v.Recall != nil || k2v.Explanation != nil || k2v.NextDue != nil || k2v.Claims == nil {
		t.Fatalf("card 2: %+v", k2v)
	}
	m := v.Mistakes[0]
	if m.SessionID != sess.ID || m.ConceptID != k1.ID || m.ClaimID == nil || *m.ClaimID != c1.ID || m.StudentText != "it is 3x" ||
		m.Label != domain.LabelContradicted || m.Quote != quote || m.SourceID != blocks[0].SourceID ||
		m.Page == nil || *m.Page != 7 || m.PageID == nil || *m.PageID != pages[0].ID {
		t.Fatalf("mistake: %+v", m)
	}

	st, err := s.StudyState(bg, a.ID, now.Add(-36*time.Hour))
	mustOK(t, err)
	if st.User.ID != a.ID || len(st.Courses) != 1 {
		t.Fatalf("state: %+v", st)
	}
	cs := st.Courses[0]
	if len(cs.Concepts) != 2 || cs.Concepts[0].ConceptID != k1.ID || !cs.Concepts[0].HasCards || !cs.Concepts[0].Taught ||
		cs.Concepts[0].DueAt == nil || cs.Concepts[1].DueAt != nil || cs.Concepts[1].Taught || cs.Concepts[1].HasCards {
		t.Fatalf("concepts: %+v", cs.Concepts)
	}
	if len(cs.Events) != 2 || cs.Events[0].Grade == nil || *cs.Events[0].Grade != domain.GradeAgain || cs.Events[1].EventType != domain.EventTeachBack {
		t.Fatalf("events since: %+v", cs.Events)
	}
	if st2, _ := s.StudyState(bg, a.ID, now.Add(time.Hour)); len(st2.Courses[0].Events) != 0 || st2.Courses[0].Events == nil {
		t.Fatalf("no events later: %+v", st2.Courses[0].Events)
	}
	// Archived courses leave the planner's view.
	mustOK(t, s.ArchiveCourse(bg, a.ID, c.ID))
	if st3, _ := s.StudyState(bg, a.ID, time.Time{}); len(st3.Courses) != 0 || st3.Courses == nil {
		t.Fatalf("archived in state: %+v", st3.Courses)
	}
}
