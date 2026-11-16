package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CrisH2307/repetition-study/internal/auth"
	"github.com/CrisH2307/repetition-study/internal/domain"
	"github.com/CrisH2307/repetition-study/internal/fakes"
	"github.com/CrisH2307/repetition-study/internal/llm"
	"github.com/CrisH2307/repetition-study/internal/store"
)

// student is one signed-in user of the real stack.
type student struct {
	e    *env
	h    http.Handler
	user domain.User
}

// newStudent invites and signs in a user through the real invite and redeem routes.
func (e *env) newStudent(name string) *student {
	e.t.Helper()
	tok, err := (&auth.Service{Store: store.New(e.pool), Clock: e.clock}).NewInvite(context.Background(), name)
	if err != nil {
		e.t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	raw := NewHandler(e.d)
	raw.ServeHTTP(rec, httptest.NewRequest("POST", "/v1/auth/redeem", strings.NewReader(fmt.Sprintf(`{"token":%q}`, tok))))
	if rec.Code != 200 || len(rec.Result().Cookies()) != 1 {
		e.t.Fatalf("redeem: %d %s", rec.Code, rec.Body)
	}
	c := rec.Result().Cookies()[0]
	if !c.HttpOnly || !c.Secure {
		e.t.Errorf("session cookie must be HttpOnly and Secure by default: %+v", c)
	}
	s := &student{e: e, h: signedIn{raw, c.Value}}
	json.Unmarshal(rec.Body.Bytes(), &s.user)
	return s
}

func (s *student) call(method, path, body string, want int, out any) {
	s.e.t.Helper()
	rec := do(s.h, method, path, body)
	if rec.Code != want {
		s.e.t.Fatalf("%s %s as %s: got %d, want %d: %s", method, path, s.user.Name, rec.Code, want, rec.Body)
	}
	if out != nil {
		if err := json.Unmarshal(rec.Body.Bytes(), out); err != nil {
			s.e.t.Fatalf("%s %s: %v: %s", method, path, err, rec.Body)
		}
	}
}

// notes is what a student's notes say. Each student gets one marker word the other never sees.
func notes(marker string) string {
	return "The mean is the sum divided by the count. " + marker + " is only in these notes."
}

// course is a ready course: one page marked Ready, one concept, one required claim.
type ready struct {
	ID, UploadID, PageID, BlockID, ConceptID, ClaimID string
}

func (s *student) makeCourse(name, marker string) ready {
	s.e.t.Helper()
	var c ready
	var sub domain.Subject
	s.call("POST", "/v1/courses", fmt.Sprintf(`{"name":%q,"weekly_hours":3,"level":"beginner"}`, name), 201, &sub)
	c.ID = sub.ID
	rec := upload(s.h, "/v1/courses/"+c.ID+"/uploads", "notes.txt", notes(marker))
	if rec.Code != 202 {
		s.e.t.Fatalf("upload: %d %s", rec.Code, rec.Body)
	}
	var up domain.Upload
	json.Unmarshal(rec.Body.Bytes(), &up)
	if up.Status != domain.UploadReading {
		s.e.t.Errorf("a new upload is %q, want reading", up.Status)
	}
	c.UploadID = up.ID
	waitUpload(s.e.t, s.h, up.ID, domain.UploadNeedsChecking)
	var pages []domain.Page
	s.call("GET", "/v1/uploads/"+up.ID+"/pages", "", 200, &pages)
	c.PageID = pages[0].ID
	s.call("POST", "/v1/pages/"+c.PageID+"/ready", "", 200, nil)
	var blocks []domain.SourceBlock
	s.call("GET", "/v1/courses/"+c.ID+"/blocks", "", 200, &blocks)
	c.BlockID = blocks[0].ID
	var con domain.Concept
	s.call("POST", "/v1/courses/"+c.ID+"/concepts", `{"name":"Mean"}`, 201, &con)
	c.ConceptID = con.ID
	var cl domain.Claim
	s.call("POST", "/v1/concepts/"+con.ID+"/claims", fmt.Sprintf(
		`{"text":"The mean is the sum over the count","kind":"definition","required":true,"anchor":{"block_id":%q,"quote":"the sum divided by the count"}}`, c.BlockID), 201, &cl)
	c.ClaimID = cl.ID
	return c
}

// Every endpoint that takes an id answers 404 for another student's id, for
// every method, and the owner's data is untouched afterwards.
func TestIsolation(t *testing.T) {
	e := newEnv(t, &fakes.LLM{Reply: "Why?"}, &fakes.LLM{JSONReply: `{"results":[]}`}, "")
	a, b := e.newStudent("Ana"), e.newStudent("Bo")
	bc := b.makeCourse("Bo's course", "Bomarker")
	var bs domain.TeachSession
	b.call("POST", "/v1/teach/sessions", fmt.Sprintf(`{"concept_id":%q}`, bc.ConceptID), 201, &bs)
	var bCard string
	if err := e.pool.QueryRow(context.Background(), `INSERT INTO cards (concept_id, front, back) VALUES ($1, 'f', 'b') RETURNING id::text`, bc.ConceptID).Scan(&bCard); err != nil {
		t.Fatal(err)
	}
	// Ana's own valid anchor: a claim edit is refused for ownership, not for its quote.
	ac := a.makeCourse("Ana's course", "Anamarker")
	claim := fmt.Sprintf(`{"text":"x","kind":"definition","required":true,"anchor":{"block_id":%q,"quote":"the sum divided by the count"}}`, ac.BlockID)
	sess := "/v1/teach/sessions/" + bs.ID
	for _, c := range []struct{ method, path, body string }{
		{"GET", "/v1/courses/" + bc.ID, ""},
		{"PATCH", "/v1/courses/" + bc.ID, `{"name":"Mine now","weekly_hours":1,"level":"beginner"}`},
		{"DELETE", "/v1/courses/" + bc.ID, ""},
		{"GET", "/v1/courses/" + bc.ID + "/view", ""},
		{"POST", "/v1/courses/" + bc.ID + "/concepts", `{"name":"Sneaky"}`},
		{"GET", "/v1/courses/" + bc.ID + "/blocks", ""},
		{"GET", "/v1/courses/" + bc.ID + "/uploads", ""},
		{"POST", "/v1/courses/" + bc.ID + "/uploads", ""}, // a form, below
		{"GET", "/v1/uploads/" + bc.UploadID, ""},
		{"GET", "/v1/uploads/" + bc.UploadID + "/pages", ""},
		{"GET", "/v1/pages/" + bc.PageID, ""},
		{"PATCH", "/v1/pages/" + bc.PageID, `{"edited_text":"changed"}`},
		{"GET", "/v1/pages/" + bc.PageID + "/image", ""},
		{"POST", "/v1/pages/" + bc.PageID + "/ready", ""},
		{"POST", "/v1/pages/" + bc.PageID + "/unready", ""},
		{"GET", "/v1/concepts/" + bc.ConceptID + "/claims", ""},
		{"POST", "/v1/concepts/" + bc.ConceptID + "/claims", claim},
		{"PATCH", "/v1/claims/" + bc.ClaimID, claim},
		{"DELETE", "/v1/claims/" + bc.ClaimID, ""},
		{"POST", "/v1/teach/sessions", fmt.Sprintf(`{"concept_id":%q}`, bc.ConceptID)},
		{"POST", sess + "/turns", `{"transcript":"hello"}`},
		{"POST", sess + "/finish", ""},
		{"GET", sess + "/report", ""},
		{"POST", sess + "/self-grades", fmt.Sprintf(`{"grades":[{"claim_id":%q,"rating":"right"}]}`, bc.ClaimID)},
		{"POST", sess + "/disputes", `{"note":"mine"}`},
		{"POST", "/v1/cards/" + bCard + "/review", `{"grade":"good"}`},
		{"POST", "/v1/concepts/" + bc.ConceptID + "/override", `{"due_at":"2030-01-01T00:00:00Z","reason":"x"}`},
	} {
		var rec *httptest.ResponseRecorder
		if c.path == "/v1/courses/"+bc.ID+"/uploads" && c.method == "POST" {
			rec = upload(a.h, c.path, "x.txt", "hello")
		} else {
			rec = do(a.h, c.method, c.path, c.body)
		}
		if rec.Code != 404 {
			t.Errorf("%s %s as the other student: got %d, want 404: %s", c.method, c.path, rec.Code, rec.Body)
		}
		if strings.Contains(rec.Body.String(), "Bomarker") {
			t.Errorf("%s %s leaks the other student's notes: %s", c.method, c.path, rec.Body)
		}
		// Not signed in at all: 401, never data.
		if r := do(e.h401(), c.method, c.path, c.body); r.Code != 401 {
			t.Errorf("%s %s without a sign-in: got %d, want 401", c.method, c.path, r.Code)
		}
	}

	// Lists show only the caller's rows.
	var cs []domain.Subject
	a.call("GET", "/v1/courses", "", 200, &cs)
	var subs []domain.Subject
	a.call("GET", "/v1/subjects", "", 200, &subs)
	if len(cs) != 1 || cs[0].ID != ac.ID || len(subs) != 1 || subs[0].ID != ac.ID {
		t.Errorf("Ana sees %+v and %+v, want only her own course", cs, subs)
	}
	var due []domain.DueItem
	a.call("GET", "/v1/reviews/due", "", 200, &due)
	for _, d := range due {
		if d.ConceptID == bc.ConceptID {
			t.Errorf("Ana's due list has Bo's concept")
		}
	}
	var today domain.Today
	a.call("GET", "/v1/today", "", 200, &today)
	if len(today.Courses) != 1 || today.Courses[0].SubjectID != ac.ID {
		t.Errorf("Ana's Today = %+v, want only her own course", today.Courses)
	}

	// Bo still owns everything, unchanged.
	var got domain.Subject
	b.call("GET", "/v1/courses/"+bc.ID, "", 200, &got)
	if got.Name != "Bo's course" {
		t.Errorf("Bo's course is now %q", got.Name)
	}
	var cl []domain.Claim
	b.call("GET", "/v1/concepts/"+bc.ConceptID+"/claims", "", 200, &cl)
	if len(cl) != 1 {
		t.Errorf("Bo has %d claims, want 1", len(cl))
	}
}

// h401 is the API with no cookie.
func (e *env) h401() http.Handler { return NewHandler(e.d) }

// The whole Phase 1 loop on the real stack with the fake model: upload, page
// check, claims, teach, check, report, self-grades, dispute, Today. The
// checker sees only this student's Ready notes.
func TestFullLoopV3(t *testing.T) {
	student := &fakes.LLM{Reply: "Why do you divide by the count?"}
	checker := &fakes.LLM{}
	e := newEnv(t, student, checker, "")
	other := e.newStudent("Bo")
	other.makeCourse("Bo's course", "Bomarker")
	s := e.newStudent("Ana")
	c := s.makeCourse("Stats", "Anamarker")

	// A claim must quote a Ready page. Un-ready is refused while a claim quotes it.
	s.call("POST", "/v1/concepts/"+c.ConceptID+"/claims", fmt.Sprintf(
		`{"text":"x","kind":"definition","required":true,"anchor":{"block_id":%q,"quote":"the median is the middle"}}`, c.BlockID), 400, nil)
	s.call("POST", "/v1/pages/"+c.PageID+"/unready", "", 400, nil)

	var ts domain.TeachSession
	s.call("POST", "/v1/teach/sessions", fmt.Sprintf(`{"concept_id":%q}`, c.ConceptID), 201, &ts)
	body, _ := json.Marshal(map[string]string{"transcript": "The mean is the sum times the count."})
	rec := do(s.h, "POST", "/v1/teach/sessions/"+ts.ID+"/turns", string(body))
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "event: done") {
		t.Fatalf("turn: %d %s", rec.Code, rec.Body)
	}

	// The checker contradicts the claim with a quote from Ana's own page.
	reply, _ := json.Marshal(map[string]any{"results": []domain.ClaimResult{{
		ClaimID: &c.ClaimID, StudentText: "The mean is the sum times the count.", Label: domain.LabelContradicted,
		Citation:   &domain.Citation{SourceID: sourceOf(t, s, c), Quote: "the sum divided by the count", Locator: "p.1"},
		Correction: ptr("It is the sum divided by the count."),
	}}})
	checker.JSONReply = string(reply)
	var rep domain.Report
	s.call("POST", "/v1/teach/sessions/"+ts.ID+"/finish", "", 200, &rep)
	if rep.Status != domain.ReportChecked || rep.Schedule == nil {
		t.Fatalf("report %q, schedule %v: want checked with a schedule", rep.Status, rep.Schedule)
	}
	if len(rep.Results) == 0 || rep.Results[0].Label != domain.LabelContradicted || rep.Results[0].Citation == nil {
		t.Errorf("results = %+v, want a contradicted claim with a verified citation", rep.Results)
	}
	var seen strings.Builder
	for _, r := range checker.Requests {
		seen.WriteString(r.System)
		for _, m := range r.Messages {
			seen.WriteString(m.Content)
		}
	}
	if !strings.Contains(seen.String(), "Anamarker") || strings.Contains(seen.String(), "Bomarker") {
		t.Errorf("the checker must see Ana's Ready notes and never Bo's")
	}
	for _, r := range student.Requests {
		all := r.System
		for _, m := range r.Messages {
			all += m.Content
		}
		if strings.Contains(all, "sum divided by the count") || strings.Contains(all, "Anamarker") {
			t.Errorf("the AI student saw her notes: %q", all)
		}
	}

	s.call("POST", "/v1/teach/sessions/"+ts.ID+"/self-grades", fmt.Sprintf(`{"grades":[{"claim_id":%q,"rating":"wrong"}]}`, c.ClaimID), 204, nil)
	s.call("POST", "/v1/teach/sessions/"+ts.ID+"/disputes", fmt.Sprintf(`{"claim_id":%q,"note":"I think this check is wrong"}`, c.ClaimID), 201, nil)
	if n := e.count(`SELECT count(*) FROM review_events WHERE event_type = 'teach_back' AND concept_id = $1`, c.ConceptID); n != 1 {
		t.Errorf("%d teach_back events, want 1: self-grades and disputes write none", n)
	}

	var view struct {
		Mistakes []domain.Mistake `json:"mistakes"`
		Notes    []domain.Upload  `json:"notes"`
	}
	s.call("GET", "/v1/courses/"+c.ID+"/view", "", 200, &view)
	if len(view.Mistakes) != 1 || len(view.Notes) != 1 || view.Notes[0].Status != domain.UploadNeedsChecking && view.Notes[0].Status != domain.UploadReady {
		t.Errorf("course view = %+v", view)
	}
	var today domain.Today
	s.call("GET", "/v1/today", "", 200, &today)
	var week domain.WeekPlan
	s.call("GET", "/v1/week?week=2026-11-09", "", 200, &week)
	if len(today.Courses) != 1 || len(week.Days) != 7 {
		t.Errorf("today courses %d, week days %d", len(today.Courses), len(week.Days))
	}

	// cmd/export reads the dispute.
	ds, err := store.New(e.pool).AllDisputes(context.Background())
	if err != nil || len(ds) != 1 || ds[0].UserID != s.user.ID {
		t.Errorf("disputes = %+v, %v", ds, err)
	}
}

func sourceOf(t *testing.T, s *student, c ready) string {
	t.Helper()
	var bs []domain.SourceBlock
	s.call("GET", "/v1/courses/"+c.ID+"/blocks", "", 200, &bs)
	return bs[0].SourceID
}

// When the monthly cap is reached the checker is not called: the report is
// unchecked and no date moves (D33).
func TestSpendCapGivesUncheckedReport(t *testing.T) {
	checker := &fakes.LLM{JSONReply: `{"results":[]}`}
	spent := 0.0
	capped := &capLLM{LLM: checker, cap: 5, total: func() float64 { return spent }, now: time.Now}
	capped.spent() // the month starts here, with nothing spent
	e := newEnv(t, &fakes.LLM{Reply: "Why?"}, capped, "")
	s := e.newStudent("Ana")
	c := s.makeCourse("Stats", "Anamarker")
	var ts domain.TeachSession
	s.call("POST", "/v1/teach/sessions", fmt.Sprintf(`{"concept_id":%q}`, c.ConceptID), 201, &ts)
	do(s.h, "POST", "/v1/teach/sessions/"+ts.ID+"/turns", `{"transcript":"The mean is the sum."}`)
	spent = 5 // the cap is reached before the check
	var rep domain.Report
	s.call("POST", "/v1/teach/sessions/"+ts.ID+"/finish", "", 200, &rep)
	if rep.Status != domain.ReportUnchecked || rep.Schedule != nil {
		t.Errorf("status %q, schedule %v: want unchecked and no schedule", rep.Status, rep.Schedule)
	}
	if len(checker.Requests) != 0 {
		t.Errorf("the checker was called %d times past the cap", len(checker.Requests))
	}
}

func TestCapLLMResetsEachMonth(t *testing.T) {
	spend, now := 0.0, time.Date(2026, 11, 30, 0, 0, 0, 0, time.UTC)
	c := &capLLM{LLM: &fakes.LLM{JSONReply: `{}`}, cap: 2, total: func() float64 { return spend }, now: func() time.Time { return now }}
	call := func() error { return c.JSON(context.Background(), domain.LLMRequest{}, &struct{}{}) }
	if err := call(); err != nil {
		t.Fatal(err)
	}
	spend = 2.5
	if err := call(); err != llm.ErrBudget {
		t.Fatalf("over the cap: %v, want ErrBudget", err)
	}
	now = now.AddDate(0, 0, 2) // December: the month's spend starts again from here
	if err := call(); err != nil {
		t.Fatalf("new month: %v", err)
	}
	spend = 5
	if err := call(); err != llm.ErrBudget {
		t.Fatalf("over the cap again: %v", err)
	}
}

// A form with both a file and text, a non-PDF, and a missing name are refused in plain words.
func TestUploadRefusals(t *testing.T) {
	st := newFake()
	var course domain.Subject
	json.Unmarshal(do(st.h, "POST", "/v1/courses", `{"name":"S","weekly_hours":1,"level":"beginner"}`).Body.Bytes(), &course)
	post := func(fields map[string]string, file string) int {
		var b bytes.Buffer
		mw := multipart.NewWriter(&b)
		for k, v := range fields {
			mw.WriteField(k, v)
		}
		if file != "" {
			fw, _ := mw.CreateFormFile("file", "a.pdf")
			fw.Write([]byte(file))
		}
		mw.Close()
		req := httptest.NewRequest("POST", "/v1/courses/"+course.ID+"/uploads", &b)
		req.Header.Set("Content-Type", mw.FormDataContentType())
		rec := httptest.NewRecorder()
		st.h.ServeHTTP(rec, req)
		return rec.Code
	}
	for name, c := range map[string]struct {
		fields map[string]string
		file   string
	}{
		"both":      {map[string]string{"filename": "a"}, "%PDF-1.4"},
		"neither":   {map[string]string{"filename": "a"}, ""},
		"no name":   {map[string]string{"text": "hi"}, ""},
		"not a pdf": {map[string]string{"filename": "a"}, "hello"},
	} {
		if name == "both" {
			c.fields["text"] = "hi"
		}
		if got := post(c.fields, c.file); got != 400 {
			t.Errorf("%s: got %d, want 400", name, got)
		}
	}
	if got := post(map[string]string{"filename": "a"}, "%PDF-1.4 ok"); got != 202 {
		t.Errorf("a PDF: got %d, want 202", got)
	}
}
