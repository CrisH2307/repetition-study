package app

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/getkin/kin-openapi/openapi3"

	"github.com/CrisH2307/repetition-study/internal/domain"
	"github.com/CrisH2307/repetition-study/internal/fakes"
)

type memSubjects struct{ list []domain.Subject }

func (m *memSubjects) List(context.Context, string) ([]domain.Subject, error) { return m.list, nil }
func (m *memSubjects) Create(_ context.Context, _ string, s domain.Subject) (domain.Subject, error) {
	s.ID = "sub-1"
	m.list = append(m.list, s)
	return s, nil
}

// allowAll is an Owner that says yes: the fake teach engine and scheduler know no users.
type allowAll struct{}

func (allowAll) Concept(context.Context, string, string) error { return nil }
func (allowAll) Card(context.Context, string, string) error    { return nil }
func (allowAll) ConceptIDs(context.Context, string) (map[string]bool, error) {
	return map[string]bool{"concept-1": true}, nil
}

// signedIn adds the session cookie to requests that carry none.
type signedIn struct {
	http.Handler
	token string
}

func (s signedIn) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if _, err := r.Cookie("session"); err != nil && s.token != "" {
		r.AddCookie(&http.Cookie{Name: "session", Value: s.token})
	}
	s.Handler.ServeHTTP(w, r)
}

// fakeStack is the whole API over the in-memory fakes, with one signed-in user.
type fakeStack struct {
	h       http.Handler
	raw     http.Handler // no cookie added
	auth    *fakes.Auth
	reviews *fakes.ReviewStore
	imports *fakes.ImportStore
	privacy *fakes.PrivacyStore
	user    domain.User
	token   string
}

func newFake() *fakeStack {
	clock := fakes.NewClock()
	auth := &fakes.Auth{Clock: clock}
	courses := &fakes.CourseStore{}
	reviews := &fakes.ReviewStore{Sessions: map[string]string{}}
	imports := &fakes.ImportStore{Owners: map[string]string{}}
	privacy := &fakes.PrivacyStore{Exports: map[string]domain.UserExport{}}
	raw := NewHandler(Deps{
		Teach: &fakes.TeachEngine{}, Sched: &fakes.Scheduler{Clock: clock}, Planner: fakes.Planner{},
		Subjects: courseSubjects{courses}, Clock: clock,
		Auth: auth, Course: courses, Reviews: reviews, Notes: fakes.CourseNotes{Store: courses},
		Today: fakes.TodayPlanner{}, Own: allowAll{},
		Feedback: &fakes.FeedbackStore{}, Privacy: privacy, Imports: imports, Importer: &fakes.Importer{Store: imports},
	})
	st := &fakeStack{raw: raw, auth: auth, reviews: reviews, imports: imports, privacy: privacy}
	st.user, st.token = st.signIn("Ana")
	st.h = signedIn{raw, st.token}
	privacy.Exports[st.user.ID] = domain.UserExport{User: st.user}
	return st
}

// signIn makes a new user and returns their session token.
func (f *fakeStack) signIn(name string) (domain.User, string) {
	inv, _ := f.auth.NewInvite(context.Background(), name)
	u, tok, _, err := f.auth.Redeem(context.Background(), inv)
	if err != nil {
		panic(err)
	}
	return u, tok
}

func fakeHandler() http.Handler { return newFake().h }

// notServed lists spec operations Phase 0 does not serve (P0-INT skips digest and plan editing).
var notServed = map[string]bool{
	"GET /v1/plan": true, "PATCH /v1/plan/blocks/{id}": true, "PATCH /v1/cards/{id}": true, "GET /v1/digest/today": true,
	// Contract v5 (Phase 3). P3-API serves these and removes them here.
	"POST /v1/courses/{id}/share": true, "DELETE /v1/shares/{id}": true,
	"GET /v1/shared/{token}": true, "POST /v1/shared/{token}/import": true,
}

func loadSpec(t *testing.T) *openapi3.T {
	t.Helper()
	doc, err := openapi3.NewLoader().LoadFromFile("../../api/openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	return doc
}

// check validates a response body against the schema the spec gives for that operation and status.
func check(t *testing.T, doc *openapi3.T, method, path string, status int, body []byte) {
	t.Helper()
	op := doc.Paths.Value(path).GetOperation(method)
	if op == nil {
		t.Fatalf("%s %s is not in the spec", method, path)
	}
	ref := op.Responses.Status(status)
	if ref == nil {
		t.Fatalf("%s %s: spec has no %d response", method, path, status)
	}
	if ref.Ref != "" && ref.Value.Content["application/json"] == nil {
		t.Fatalf("%s %s %d: no JSON schema", method, path, status)
	}
	var v any
	if err := json.Unmarshal(body, &v); err != nil {
		t.Fatalf("%s %s: not JSON: %v: %s", method, path, err, body)
	}
	if err := ref.Value.Content["application/json"].Schema.Value.VisitJSON(v, openapi3.EnableFormatValidation()); err != nil {
		t.Errorf("%s %s %d does not match the spec: %v\n%s", method, path, status, err, body)
	}
}

// upload posts a pasted-text upload as the web app does.
func upload(h http.Handler, path, name, text string) *httptest.ResponseRecorder {
	var b bytes.Buffer
	mw := multipart.NewWriter(&b)
	mw.WriteField("filename", name)
	mw.WriteField("text", text)
	mw.Close()
	req := httptest.NewRequest("POST", path, &b)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// waitUpload polls until the background read has set the status.
func waitUpload(t *testing.T, h http.Handler, id string, want domain.UploadStatus) domain.Upload {
	t.Helper()
	var u domain.Upload
	for i := 0; i < 200; i++ {
		json.Unmarshal(do(h, "GET", "/v1/uploads/"+id, "").Body.Bytes(), &u)
		if u.Status == want {
			return u
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("upload %s is %q, want %q", id, u.Status, want)
	return u
}

// csvUpload posts a CSV file as the web app does.
func csvUpload(h http.Handler, path, name, csv string) *httptest.ResponseRecorder {
	var b bytes.Buffer
	mw := multipart.NewWriter(&b)
	fw, _ := mw.CreateFormFile("file", name)
	fw.Write([]byte(csv))
	mw.Close()
	req := httptest.NewRequest("POST", path, &b)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func do(h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, path, strings.NewReader(body)))
	return rec
}

func TestHandlersMatchOpenAPI(t *testing.T) {
	doc := loadSpec(t)
	st := newFake()
	h := st.h
	served := map[string]bool{}
	call := func(method, tmpl, path, body string, want int) []byte {
		t.Helper()
		served[method+" "+tmpl] = true
		rec := do(h, method, path, body)
		if rec.Code != want {
			t.Fatalf("%s %s: got %d, want %d: %s", method, path, rec.Code, want, rec.Body)
		}
		if rec.Body.Len() > 0 { // 204 has no body
			check(t, doc, method, tmpl, want, rec.Body.Bytes())
		}
		return rec.Body.Bytes()
	}

	call("GET", "/healthz", "/healthz", "", 200)
	call("POST", "/v1/subjects", "/v1/subjects", `{"name":"Math","weekly_hours":4,"level":"beginner","exam_date":"2026-12-15"}`, 201)
	call("GET", "/v1/subjects", "/v1/subjects", "", 200)
	call("POST", "/v1/plan/generate", "/v1/plan/generate", `{"week_start":"2026-11-09"}`, 200)
	call("GET", "/v1/analytics/further-planning", "/v1/analytics/further-planning?week=2026-11-09", "", 200)

	var ts domain.TeachSession
	json.Unmarshal(call("POST", "/v1/teach/sessions", "/v1/teach/sessions", `{"concept_id":"concept-1"}`, 201), &ts)
	st.reviews.Sessions[ts.ID] = st.user.ID
	sid := "/v1/teach/sessions/" + ts.ID

	// SSE: token events, then done.
	served["POST /v1/teach/sessions/{id}/turns"] = true
	rec := do(h, "POST", sid+"/turns", `{"transcript":"X beats Y with chance a/(a+b)","latex":"\\frac{a}{a+b}"}`)
	if rec.Code != 200 || !strings.HasPrefix(rec.Header().Get("Content-Type"), "text/event-stream") {
		t.Fatalf("turn: %d %s", rec.Code, rec.Header())
	}
	var events []string
	sc := bufio.NewScanner(rec.Body)
	for sc.Scan() {
		line := sc.Text()
		if e, ok := strings.CutPrefix(line, "event: "); ok {
			events = append(events, e)
		}
		if d, ok := strings.CutPrefix(line, "data: "); ok {
			var m map[string]any
			if err := json.Unmarshal([]byte(d), &m); err != nil {
				t.Fatalf("bad SSE data %q", d)
			}
		}
	}
	if len(events) < 2 || events[0] != "token" || events[len(events)-1] != "done" {
		t.Fatalf("events = %v, want token... then done", events)
	}

	call("POST", "/v1/teach/sessions/{id}/finish", sid+"/finish", "", 200)
	call("GET", "/v1/teach/sessions/{id}/report", sid+"/report", "", 200)
	call("GET", "/v1/reviews/due", "/v1/reviews/due", "", 200)
	call("POST", "/v1/cards/{id}/review", "/v1/cards/card-1/review", `{"grade":"good"}`, 200)
	call("POST", "/v1/concepts/{id}/override", "/v1/concepts/concept-1/override", `{"due_at":"2026-12-01T00:00:00Z","reason":"exam moved"}`, 200)

	// Contract v3: the whole course loop over the fakes.
	call("GET", "/v1/me", "/v1/me", "", 200)
	var course domain.Subject
	json.Unmarshal(call("POST", "/v1/courses", "/v1/courses", `{"name":"Stats","weekly_hours":3,"level":"beginner"}`, 201), &course)
	cp := "/v1/courses/" + course.ID
	call("GET", "/v1/courses", "/v1/courses", "", 200)
	call("GET", "/v1/courses/{id}", cp, "", 200)
	call("PATCH", "/v1/courses/{id}", cp, `{"name":"Statistics","weekly_hours":4,"level":"intermediate","exam_date":"2026-12-15"}`, 200)
	var concept domain.Concept
	json.Unmarshal(call("POST", "/v1/courses/{id}/concepts", cp+"/concepts", `{"name":"Mean"}`, 201), &concept)

	var up domain.Upload
	rec = upload(h, cp+"/uploads", "notes.txt", "The mean is the sum divided by the count.")
	if rec.Code != 202 {
		t.Fatalf("upload: %d %s", rec.Code, rec.Body)
	}
	served["POST /v1/courses/{id}/uploads"] = true
	check(t, doc, "POST", "/v1/courses/{id}/uploads", 202, rec.Body.Bytes())
	json.Unmarshal(rec.Body.Bytes(), &up)
	call("GET", "/v1/courses/{id}/uploads", cp+"/uploads", "", 200)
	waitUpload(t, h, up.ID, domain.UploadNeedsChecking)
	call("GET", "/v1/uploads/{id}", "/v1/uploads/"+up.ID, "", 200)
	var pages []domain.Page
	json.Unmarshal(call("GET", "/v1/uploads/{id}/pages", "/v1/uploads/"+up.ID+"/pages", "", 200), &pages)
	pp := "/v1/pages/" + pages[0].ID
	call("GET", "/v1/pages/{id}", pp, "", 200)
	call("PATCH", "/v1/pages/{id}", pp, `{"edited_text":"The mean is the sum divided by the count."}`, 200)
	call("POST", "/v1/pages/{id}/ready", pp+"/ready", "", 200)
	var blocks []domain.SourceBlock
	json.Unmarshal(call("GET", "/v1/courses/{id}/blocks", cp+"/blocks", "", 200), &blocks)
	claimBody := fmt.Sprintf(`{"text":"The mean is sum over count","kind":"definition","required":true,"anchor":{"block_id":%q,"quote":"the sum divided by the count"}}`, blocks[0].ID)
	var claim domain.Claim
	json.Unmarshal(call("POST", "/v1/concepts/{id}/claims", "/v1/concepts/"+concept.ID+"/claims", claimBody, 201), &claim)
	call("GET", "/v1/concepts/{id}/claims", "/v1/concepts/"+concept.ID+"/claims", "", 200)
	call("PATCH", "/v1/claims/{id}", "/v1/claims/"+claim.ID, claimBody, 200)
	call("POST", "/v1/concepts/{id}/claims", "/v1/concepts/"+concept.ID+"/claims",
		fmt.Sprintf(`{"text":"x","kind":"definition","required":true,"anchor":{"block_id":%q,"quote":"not in the notes"}}`, blocks[0].ID), 400)
	call("DELETE", "/v1/claims/{id}", "/v1/claims/"+claim.ID, "", 204)
	call("POST", "/v1/pages/{id}/unready", pp+"/unready", "", 200)
	served["GET /v1/pages/{id}/image"] = true
	if rec := do(h, "GET", pp+"/image", ""); rec.Code != 404 { // the fake page has no picture
		t.Errorf("image of a page without one: %d", rec.Code)
	}
	st.reviews.Views = map[string]map[string]domain.CourseView{st.user.ID: {course.ID: {Course: course}}}
	call("GET", "/v1/courses/{id}/view", cp+"/view", "", 200)
	call("POST", "/v1/teach/sessions/{id}/self-grades", sid+"/self-grades", `{"grades":[{"claim_id":"c1","rating":"right"}]}`, 204)
	call("POST", "/v1/teach/sessions/{id}/self-grades", sid+"/self-grades", `{"grades":[{"claim_id":"c1","rating":"great"}]}`, 400)
	call("POST", "/v1/teach/sessions/{id}/disputes", sid+"/disputes", `{"claim_id":null,"note":"I think this is wrong"}`, 201)
	call("GET", "/v1/today", "/v1/today", "", 200)
	call("GET", "/v1/week", "/v1/week?week=2026-11-09", "", 200)
	call("GET", "/v1/week", "/v1/week?week=soon", "", 400)
	call("DELETE", "/v1/courses/{id}", cp, "", 204)
	call("GET", "/v1/courses/{id}", cp, "", 404)

	// Errors use {"error": "..."} and the statuses the spec names.
	call("POST", "/v1/subjects", "/v1/subjects", `{"name":"","weekly_hours":4,"level":"beginner"}`, 400)
	call("POST", "/v1/teach/sessions/{id}/finish", "/v1/teach/sessions/nope/finish", "", 404)
	call("GET", "/v1/teach/sessions/{id}/report", "/v1/teach/sessions/nope/report", "", 404)

	// Contract v4: consent, feedback, import, export and delete over the fakes.
	call("POST", "/v1/consent", "/v1/consent", `{"text_version":"2026-11"}`, 201)
	call("POST", "/v1/consent", "/v1/consent", `{"text_version":""}`, 400)
	var fp domain.FeedbackPrompt
	json.Unmarshal(call("GET", "/v1/feedback", "/v1/feedback", "", 200), &fp)
	if fp.Question == "" || fp.Answer != nil {
		t.Errorf("fresh prompt = %+v, want a question and no answer", fp)
	}
	call("POST", "/v1/feedback", "/v1/feedback", `{"answer":"Very disappointed"}`, 201)
	call("POST", "/v1/feedback", "/v1/feedback", `{"answer":"  "}`, 400)
	json.Unmarshal(call("GET", "/v1/feedback", "/v1/feedback", "", 200), &fp)
	if fp.Answer == nil || *fp.Answer != "Very disappointed" {
		t.Errorf("answered prompt = %+v", fp)
	}
	var c2 domain.Subject
	json.Unmarshal(call("POST", "/v1/courses", "/v1/courses", `{"name":"Imports","weekly_hours":1,"level":"beginner"}`, 201), &c2)
	st.imports.Owners[c2.ID] = st.user.ID
	var job domain.ImportJob
	rec = csvUpload(h, "/v1/courses/"+c2.ID+"/import", "claims.csv", "concept,claim,kind,quote\nMean,The mean is sum over count,definition,the sum divided by the count\n")
	if rec.Code != 202 {
		t.Fatalf("import: %d %s", rec.Code, rec.Body)
	}
	served["POST /v1/courses/{id}/import"] = true
	check(t, doc, "POST", "/v1/courses/{id}/import", 202, rec.Body.Bytes())
	json.Unmarshal(rec.Body.Bytes(), &job)
	call("GET", "/v1/imports/{id}", "/v1/imports/"+job.ID, "", 200)
	call("GET", "/v1/me/export", "/v1/me/export", "", 200)
	if cd := do(h, "GET", "/v1/me/export", "").Header().Get("Content-Disposition"); !strings.HasPrefix(cd, "attachment") {
		t.Errorf("export Content-Disposition = %q, want a download", cd)
	}
	call("DELETE", "/v1/me", "/v1/me", `{"confirm":"yes"}`, 400)
	rec = do(h, "DELETE", "/v1/me", `{"confirm":"delete my account"}`)
	served["DELETE /v1/me"] = true
	if rec.Code != 204 || len(st.privacy.Deleted) != 1 {
		t.Fatalf("delete me: %d, deleted %v", rec.Code, st.privacy.Deleted)
	}
	if cs := rec.Result().Cookies(); len(cs) != 1 || cs[0].MaxAge >= 0 {
		t.Errorf("delete me must clear the cookie, got %+v", cs)
	}

	// Sign-out ends the session; redeem and healthz are the only open routes.
	call("POST", "/v1/auth/sign-out", "/v1/auth/sign-out", "", 204)
	call("GET", "/v1/me", "/v1/me", "", 401)
	call("GET", "/healthz", "/healthz", "", 200)
	inv, _ := st.auth.NewInvite(context.Background(), "Bo")
	rec = do(h, "POST", "/v1/auth/redeem", fmt.Sprintf(`{"token":%q}`, inv))
	served["POST /v1/auth/redeem"] = true
	if rec.Code != 200 || rec.Result().Cookies()[0].Name != "session" {
		t.Fatalf("redeem: %d %s", rec.Code, rec.Body)
	}
	check(t, doc, "POST", "/v1/auth/redeem", 200, rec.Body.Bytes())

	// Every operation in the spec is either served or on the known skip list.
	for path, item := range doc.Paths.Map() {
		for method := range item.Operations() {
			key := method + " " + path
			if !served[key] && !notServed[key] {
				t.Errorf("%s is in the spec but no test calls it", key)
			}
			if served[key] && notServed[key] {
				t.Errorf("%s is served now: drop it from notServed", key)
			}
		}
	}
	// Routes outside the spec do not exist.
	for _, k := range []string{"GET /v1/digest/today", "GET /v1/plan?week=2026-11-09"} {
		m, p, _ := strings.Cut(k, " ")
		if rec := do(h, m, p, ""); rec.Code < 400 {
			t.Errorf("%s answered %d, want an error", k, rec.Code)
		}
	}
}

func TestBadInput(t *testing.T) {
	st := newFake()
	h := st.h
	st.reviews.Sessions["x"] = st.user.ID
	for _, c := range []struct{ method, path, body string }{
		{"POST", "/v1/teach/sessions", `{}`},
		{"POST", "/v1/teach/sessions", `not json`},
		{"POST", "/v1/teach/sessions/x/turns", `{"transcript":"  "}`},
		{"POST", "/v1/cards/x/review", `{"grade":"perfect"}`},
		{"POST", "/v1/concepts/x/override", `{"due_at":"2026-12-01T00:00:00Z","reason":" "}`},
		{"POST", "/v1/plan/generate", `{"week_start":"Monday"}`},
		{"GET", "/v1/analytics/further-planning", ``},
	} {
		if rec := do(h, c.method, c.path, c.body); rec.Code != 400 {
			t.Errorf("%s %s %s: got %d, want 400", c.method, c.path, c.body, rec.Code)
		}
	}
	if rec := do(h, "POST", "/v1/teach/sessions/nope/turns", `{"transcript":"hi"}`); rec.Code != 404 {
		t.Errorf("turn on unknown session: got %d, want 404", rec.Code)
	}
}

type failingEngine struct {
	*fakes.TeachEngine
	err error
}

func (f failingEngine) Finish(context.Context, string) (domain.Report, error) {
	return domain.Report{}, f.err
}

// A caller that hangs up or runs out of time mid-Finish is not a server
// error: no 500 and no error log line.
func TestFinishCallerGaveUp(t *testing.T) {
	for _, c := range []struct {
		err  error
		code int
	}{
		{context.Canceled, 499},
		{fmt.Errorf("teach: %w", context.DeadlineExceeded), 504},
	} {
		var logs bytes.Buffer
		clock := fakes.NewClock()
		h := NewHandler(Deps{Teach: failingEngine{&fakes.TeachEngine{}, c.err}, Sched: &fakes.Scheduler{Clock: clock},
			Planner: fakes.Planner{}, Subjects: &memSubjects{}, Clock: clock, Log: slog.New(slog.NewTextHandler(&logs, nil))})
		rec := do(h, "POST", "/v1/teach/sessions/x/finish", "")
		if rec.Code != c.code || !strings.Contains(rec.Body.String(), `"error"`) {
			t.Errorf("%v: got %d %s, want %d with an error body", c.err, rec.Code, rec.Body, c.code)
		}
		if l := logs.String(); !strings.Contains(l, "level=INFO") || strings.Contains(l, "level=ERROR") {
			t.Errorf("%v: want one quiet info line, got %q", c.err, l)
		}
	}
}
