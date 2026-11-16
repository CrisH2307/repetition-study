package app

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"

	"github.com/CrisH2307/repetition-study/internal/domain"
	"github.com/CrisH2307/repetition-study/internal/fakes"
)

type memSubjects struct{ list []domain.Subject }

func (m *memSubjects) List(context.Context) ([]domain.Subject, error) { return m.list, nil }
func (m *memSubjects) Create(_ context.Context, s domain.Subject) (domain.Subject, error) {
	s.ID = "sub-1"
	m.list = append(m.list, s)
	return s, nil
}

func fakeHandler() http.Handler {
	clock := fakes.NewClock()
	eng := &fakes.TeachEngine{}
	return NewHandler(Deps{
		Teach: eng, Sched: &fakes.Scheduler{Clock: clock}, Planner: fakes.Planner{},
		Subjects: &memSubjects{}, Clock: clock,
	})
}

// notServed lists spec operations Phase 0 does not serve (P0-INT skips digest and plan editing).
var notServed = map[string]bool{
	"GET /v1/plan": true, "PATCH /v1/plan/blocks/{id}": true, "PATCH /v1/cards/{id}": true, "GET /v1/digest/today": true,
	// Contract v3. P1-API serves these.
	"DELETE /v1/claims/{id}":                   true,
	"DELETE /v1/courses/{id}":                  true,
	"GET /v1/concepts/{id}/claims":             true,
	"GET /v1/courses":                          true,
	"GET /v1/courses/{id}":                     true,
	"GET /v1/courses/{id}/blocks":              true,
	"GET /v1/courses/{id}/uploads":             true,
	"GET /v1/courses/{id}/view":                true,
	"GET /v1/me":                               true,
	"GET /v1/pages/{id}":                       true,
	"GET /v1/pages/{id}/image":                 true,
	"GET /v1/today":                            true,
	"GET /v1/uploads/{id}":                     true,
	"GET /v1/uploads/{id}/pages":               true,
	"GET /v1/week":                             true,
	"PATCH /v1/claims/{id}":                    true,
	"PATCH /v1/courses/{id}":                   true,
	"PATCH /v1/pages/{id}":                     true,
	"POST /v1/auth/redeem":                     true,
	"POST /v1/auth/sign-out":                   true,
	"POST /v1/concepts/{id}/claims":            true,
	"POST /v1/courses":                         true,
	"POST /v1/courses/{id}/concepts":           true,
	"POST /v1/courses/{id}/uploads":            true,
	"POST /v1/pages/{id}/ready":                true,
	"POST /v1/pages/{id}/unready":              true,
	"POST /v1/teach/sessions/{id}/disputes":    true,
	"POST /v1/teach/sessions/{id}/self-grades": true,
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

func do(h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, path, strings.NewReader(body)))
	return rec
}

func TestHandlersMatchOpenAPI(t *testing.T) {
	doc := loadSpec(t)
	h := fakeHandler()
	served := map[string]bool{}
	call := func(method, tmpl, path, body string, want int) []byte {
		t.Helper()
		served[method+" "+tmpl] = true
		rec := do(h, method, path, body)
		if rec.Code != want {
			t.Fatalf("%s %s: got %d, want %d: %s", method, path, rec.Code, want, rec.Body)
		}
		check(t, doc, method, tmpl, want, rec.Body.Bytes())
		return rec.Body.Bytes()
	}

	call("GET", "/healthz", "/healthz", "", 200)
	call("POST", "/v1/subjects", "/v1/subjects", `{"name":"Math","weekly_hours":4,"level":"beginner","exam_date":"2026-12-15"}`, 201)
	call("GET", "/v1/subjects", "/v1/subjects", "", 200)
	call("POST", "/v1/plan/generate", "/v1/plan/generate", `{"week_start":"2026-11-09"}`, 200)
	call("GET", "/v1/analytics/further-planning", "/v1/analytics/further-planning?week=2026-11-09", "", 200)

	var ts domain.TeachSession
	json.Unmarshal(call("POST", "/v1/teach/sessions", "/v1/teach/sessions", `{"concept_id":"concept-1"}`, 201), &ts)
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

	// Errors use {"error": "..."} and the statuses the spec names.
	call("POST", "/v1/subjects", "/v1/subjects", `{"name":"","weekly_hours":4,"level":"beginner"}`, 400)
	call("POST", "/v1/teach/sessions/{id}/finish", "/v1/teach/sessions/nope/finish", "", 404)
	call("GET", "/v1/teach/sessions/{id}/report", "/v1/teach/sessions/nope/report", "", 404)

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
	h := fakeHandler()
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
