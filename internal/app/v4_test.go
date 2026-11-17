package app

import (
	"context"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CrisH2307/repetition-study/internal/domain"
	"github.com/CrisH2307/repetition-study/internal/fakes"
	"github.com/CrisH2307/repetition-study/internal/store"
)

// Every v4 endpoint is closed to other students (404) and to anyone not signed in (401).
func TestIsolationV4(t *testing.T) {
	e := newEnv(t, &fakes.LLM{Reply: "Why?"}, &fakes.LLM{JSONReply: `{"results":[]}`}, "")
	a, b := e.newStudent("Ana"), e.newStudent("Bo")
	bc := b.makeCourse("Bo's course", "Bomarker")
	rec := csvUpload(b.h, "/v1/courses/"+bc.ID+"/import", "c.csv", "concept,claim,kind,quote\nMean,The mean is sum over count,definition,the sum divided by the count\n")
	if rec.Code != 202 {
		t.Fatalf("Bo's import: %d %s", rec.Code, rec.Body)
	}
	var job domain.ImportJob
	b.call("GET", "/v1/imports/"+jobID(rec), "", 200, &job)
	b.call("POST", "/v1/consent", `{"text_version":"v1"}`, 201, nil)
	b.call("POST", "/v1/feedback", `{"answer":"Bomarker answer"}`, 201, nil)

	for _, c := range []struct{ method, path, body string }{
		{"POST", "/v1/courses/" + bc.ID + "/import", ""}, // a form, below
		{"GET", "/v1/imports/" + job.ID, ""},
	} {
		var r *httptest.ResponseRecorder
		if c.method == "POST" {
			r = csvUpload(a.h, c.path, "c.csv", "concept,claim,kind,quote\nX,y,definition,z\n")
		} else {
			r = do(a.h, c.method, c.path, c.body)
		}
		if r.Code != 404 {
			t.Errorf("%s %s as the other student: got %d, want 404: %s", c.method, c.path, r.Code, r.Body)
		}
	}
	// Endpoints with no id in the path act on the caller only.
	for _, p := range []string{"/v1/me/export", "/v1/feedback"} {
		r := do(a.h, "GET", p, "")
		if r.Code != 200 || strings.Contains(r.Body.String(), "Bomarker") || strings.Contains(r.Body.String(), b.user.ID) {
			t.Errorf("GET %s as Ana: %d, leaks Bo's data: %s", p, r.Code, r.Body)
		}
	}
	// Not signed in: 401 on all seven.
	for _, c := range []struct{ method, path string }{
		{"GET", "/v1/me/export"}, {"DELETE", "/v1/me"}, {"POST", "/v1/consent"}, {"GET", "/v1/feedback"},
		{"POST", "/v1/feedback"}, {"POST", "/v1/courses/" + bc.ID + "/import"}, {"GET", "/v1/imports/" + job.ID},
	} {
		if r := do(e.h401(), c.method, c.path, ""); r.Code != 401 {
			t.Errorf("%s %s without a sign-in: got %d, want 401", c.method, c.path, r.Code)
		}
	}
	// Ana deleting her account leaves Bo's everything.
	a.call("DELETE", "/v1/me", `{"confirm":"delete my account"}`, 204, nil)
	b.call("GET", "/v1/imports/"+job.ID, "", 200, nil)
	var got domain.UserExport
	b.call("GET", "/v1/me/export", "", 200, &got)
	if len(got.Consents) != 1 || len(got.Feedback) != 1 || len(got.Imports) != 1 {
		t.Errorf("Bo's export after Ana left: %d consents, %d feedback, %d imports, want 1 each", len(got.Consents), len(got.Feedback), len(got.Imports))
	}
}

func jobID(rec *httptest.ResponseRecorder) string {
	s := rec.Body.String()
	i := strings.Index(s, `"id":"`) + 6
	return s[i : i+strings.Index(s[i:], `"`)]
}

// Delete my account: the confirm field is required, the cookie is cleared and the old sign-in stops working.
func TestDeleteAccountEndToEnd(t *testing.T) {
	e := newEnv(t, &fakes.LLM{Reply: "Why?"}, &fakes.LLM{JSONReply: `{"results":[]}`}, "")
	a := e.newStudent("Ana")
	a.makeCourse("Stats", "Anamarker")
	a.call("DELETE", "/v1/me", `{"confirm":"yes"}`, 400, nil)
	a.call("GET", "/v1/me", "", 200, nil)
	rec := do(a.h, "DELETE", "/v1/me", `{"confirm":"delete my account"}`)
	if rec.Code != 204 {
		t.Fatalf("delete: %d %s", rec.Code, rec.Body)
	}
	if cs := rec.Result().Cookies(); len(cs) != 1 || cs[0].MaxAge >= 0 {
		t.Errorf("cookie not cleared: %+v", cs)
	}
	a.call("GET", "/v1/me", "", 401, nil)
	if n := e.count(`SELECT count(*) FROM users WHERE id = $1`, a.user.ID); n != 0 {
		t.Errorf("user row still there")
	}
}

// With the real Postgres ledger: a month at the cap makes the check report
// unchecked and no date moves, and a second cap wrapper (a restart) sees the same total.
func TestSpendCapFromPostgresLedger(t *testing.T) {
	checker := &fakes.LLM{JSONReply: `{"results":[]}`}
	var ledger domain.SpendLedger
	capped := &capLLM{LLM: checker, cap: 1, now: time.Now}
	e := newEnv(t, &fakes.LLM{Reply: "Why?"}, capped, "")
	ledger = store.New(e.pool)
	capped.ledger = ledger
	s := e.newStudent("Ana")
	c := s.makeCourse("Stats", "Anamarker")
	var ts domain.TeachSession
	s.call("POST", "/v1/teach/sessions", fmt.Sprintf(`{"concept_id":%q}`, c.ConceptID), 201, &ts)
	do(s.h, "POST", "/v1/teach/sessions/"+ts.ID+"/turns", `{"transcript":"The mean is the sum."}`)
	if _, err := ledger.Add(context.Background(), time.Now(), 1.5); err != nil {
		t.Fatal(err)
	}
	var rep domain.Report
	s.call("POST", "/v1/teach/sessions/"+ts.ID+"/finish", "", 200, &rep)
	if rep.Status != domain.ReportUnchecked || rep.Schedule != nil || len(checker.Requests) != 0 {
		t.Errorf("status %q, schedule %v, checker calls %d: want unchecked, no schedule, no calls", rep.Status, rep.Schedule, len(checker.Requests))
	}
	restarted := &capLLM{LLM: checker, cap: 1, ledger: store.New(e.pool), now: time.Now}
	if err := restarted.JSON(context.Background(), domain.LLMRequest{}, &struct{}{}); err == nil {
		t.Error("a restarted process must still see the month at the cap")
	}
}
