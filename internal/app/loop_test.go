package app

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/CrisH2307/repetition-study/internal/cite"
	"github.com/CrisH2307/repetition-study/internal/domain"
	"github.com/CrisH2307/repetition-study/internal/fakes"
	"github.com/CrisH2307/repetition-study/internal/store"
	"github.com/CrisH2307/repetition-study/migrations"
)

// Fixture ids (content/fixture/unit.json).
const (
	fxSubject  = "00000000-0000-0000-0000-0000000000f0"
	fxSource   = "00000000-0000-0000-0000-0000000000f1"
	fxConcept  = "00000000-0000-0000-0000-00000000f001"
	fxClaim1   = "00000000-0000-0000-0000-0000000f0001"
	fxClaim2   = "00000000-0000-0000-0000-0000000f0002"
	fxConcept2 = "00000000-0000-0000-0000-00000000f002"
)

// env is a real stack (store, scheduler, planner, checker, teach engine) on a
// throwaway Postgres schema, with the model clients and clock chosen by the test.
type env struct {
	t     *testing.T
	pool  *pgxpool.Pool
	d     Deps
	h     http.Handler
	clock *fakes.Clock
	pack  domain.Pack
}

func newEnv(t *testing.T, student, checkerLLM domain.LLM, checkerModel string) *env {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set: run `make up migrate`, then `make test`")
	}
	ctx := context.Background()
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	admin, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Skipf("cannot reach Postgres: %v", err)
	}
	if err := admin.Ping(ctx); err != nil {
		admin.Close()
		t.Skipf("cannot reach Postgres: %v", err)
	}
	schema := fmt.Sprintf("app_test_%d", time.Now().UnixNano())
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { admin.Exec(context.Background(), "DROP SCHEMA "+schema+" CASCADE"); admin.Close() })
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	files, _ := migrations.FS.ReadDir(".")
	var names []string
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".sql") {
			names = append(names, f.Name())
		}
	}
	sort.Strings(names)
	for _, n := range names {
		sql, _ := migrations.FS.ReadFile(n)
		if _, err := pool.Exec(ctx, string(sql)); err != nil {
			t.Fatalf("%s: %v", n, err)
		}
	}

	var pack domain.Pack
	b, err := os.ReadFile("../../content/fixture/unit.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, &pack); err != nil {
		t.Fatal(err)
	}
	if err := store.New(pool).LoadPack(ctx, pack); err != nil {
		t.Fatal(err)
	}
	clock := fakes.NewClock()
	d := Wire(pool, student, checkerLLM, checkerModel, clock, nil)
	return &env{t: t, pool: pool, d: d, h: NewHandler(d), clock: clock, pack: pack}
}

func (e *env) call(method, path, body string, want int, out any) {
	e.t.Helper()
	rec := httptest.NewRecorder()
	e.h.ServeHTTP(rec, httptest.NewRequest(method, path, strings.NewReader(body)))
	if rec.Code != want {
		e.t.Fatalf("%s %s: got %d, want %d: %s", method, path, rec.Code, want, rec.Body)
	}
	if out != nil {
		if err := json.Unmarshal(rec.Body.Bytes(), out); err != nil {
			e.t.Fatalf("%s %s: %v: %s", method, path, err, rec.Body)
		}
	}
}

func (e *env) count(sql string, args ...any) (n int) {
	e.t.Helper()
	if err := e.pool.QueryRow(context.Background(), sql, args...).Scan(&n); err != nil {
		e.t.Fatal(err)
	}
	return
}

// teachBacks counts the concept's teach_back events.
func (e *env) teachBacks() int {
	return e.count(`SELECT count(*) FROM review_events WHERE event_type = 'teach_back' AND concept_id = $1`, fxConcept)
}

// nextWeekHours reads next week's hours for the fixture subject through the API.
func (e *env) nextWeekHours() float64 {
	e.t.Helper()
	var fp domain.FurtherPlanning
	e.call("GET", "/v1/analytics/further-planning?week=2026-12-07", "", 200, &fp)
	for _, s := range fp.Subjects {
		if s.SubjectID == fxSubject {
			return s.NextWeekHours
		}
	}
	e.t.Fatal("fixture subject missing from further planning")
	return 0
}

// runTurn sends one turn and returns the SSE event names and the streamed text.
func (e *env) runTurn(sid, transcript string) (events []string, text string) {
	e.t.Helper()
	body, _ := json.Marshal(map[string]string{"transcript": transcript})
	rec := httptest.NewRecorder()
	e.h.ServeHTTP(rec, httptest.NewRequest("POST", "/v1/teach/sessions/"+sid+"/turns", strings.NewReader(string(body))))
	if rec.Code != 200 {
		e.t.Fatalf("turn: %d %s", rec.Code, rec.Body)
	}
	var ev string
	sc := bufio.NewScanner(rec.Body)
	for sc.Scan() {
		l := sc.Text()
		if v, ok := strings.CutPrefix(l, "event: "); ok {
			ev = v
			events = append(events, v)
		}
		if v, ok := strings.CutPrefix(l, "data: "); ok && ev == "token" {
			var m struct{ Text string }
			json.Unmarshal([]byte(v), &m)
			text += m.Text
		}
	}
	return
}

// plantedMistake is what the human teaches: the first claim is upside down.
const plantedMistake = "The chance that X finishes before Y is b/(a+b). The smaller of the two times is exponential with rate a+b."

// runLoop is the whole product loop. The fake-model test and the realllm test
// both run it: seed, schedule a baseline, teach with one mistake, finish twice.
func runLoop(t *testing.T, e *env) {
	ctx := context.Background()
	src := e.pack.Sources[0].Text

	// Baseline: both concepts were taught well, twice, so the planner expects
	// good recall at the exam (15 Dec). The clock ends on Monday 7 Dec.
	supported := func(claims ...string) (out []domain.ClaimResult) {
		for _, c := range claims {
			out = append(out, domain.ClaimResult{ClaimID: ptr(c), Label: domain.LabelSupported})
		}
		return
	}
	pass := func() {
		for _, c := range []struct {
			id     string
			claims []string
		}{
			{fxConcept, []string{fxClaim1, fxClaim2, "00000000-0000-0000-0000-0000000f0003"}},
			{fxConcept2, []string{"00000000-0000-0000-0000-0000000f0004", "00000000-0000-0000-0000-0000000f0005", "00000000-0000-0000-0000-0000000f0006"}},
		} {
			if _, err := e.d.Sched.ApplyTeachBack(ctx, c.id, supported(c.claims...)); err != nil {
				t.Fatal(err)
			}
		}
	}
	e.clock.Advance(21 * 24 * time.Hour) // 30 Nov
	pass()
	e.clock.Advance(3 * 24 * time.Hour) // 3 Dec
	pass()
	e.clock.Advance(4 * 24 * time.Hour) // 7 Dec
	baseline := e.clock.Now().AddDate(0, 0, 5)
	var ov domain.ReviewEvent
	e.call("POST", "/v1/concepts/"+fxConcept+"/override",
		fmt.Sprintf(`{"due_at":%q,"reason":"baseline for the loop test"}`, baseline.Format(time.RFC3339)), 200, &ov)
	e.clock.Advance(time.Hour)
	hoursBefore := e.nextWeekHours()
	eventsBefore := e.teachBacks()
	if n := e.count(`SELECT count(*) FROM cards WHERE concept_id = $1`, fxConcept); n != 0 {
		t.Fatalf("%d cards before the session, want 0", n)
	}

	var ts domain.TeachSession
	e.call("POST", "/v1/teach/sessions", fmt.Sprintf(`{"concept_id":%q}`, fxConcept), 201, &ts)
	events, reply := e.runTurn(ts.ID, plantedMistake)
	if len(events) < 2 || events[0] != "token" || events[len(events)-1] != "done" {
		t.Fatalf("turn events = %v, want token... then done", events)
	}
	if !strings.Contains(reply, "?") {
		t.Errorf("student reply %q is not a question", reply)
	}

	var rep domain.Report
	e.call("POST", "/v1/teach/sessions/"+ts.ID+"/finish", "", 200, &rep)
	if rep.Status != domain.ReportChecked || rep.Schedule == nil {
		t.Fatalf("report status %q, schedule %v: want checked with a schedule", rep.Status, rep.Schedule)
	}

	// A verified citation: the quote is an exact substring of the approved source.
	var cited bool
	for _, r := range rep.Results {
		if r.Label == domain.LabelContradicted && r.Citation != nil && r.Correction != nil && cite.Verify(src, r.Citation.Quote) {
			cited = true
		}
	}
	if !cited {
		t.Errorf("no contradicted result with a verified citation and a correction: %+v", rep.Results)
	}

	// An earlier next review, and one teach_back event.
	if rep.Schedule.EventType != domain.EventTeachBack || rep.Schedule.ConceptID != fxConcept {
		t.Errorf("schedule event = %+v", rep.Schedule)
	}
	if !rep.Schedule.DueBefore.Equal(baseline) || !rep.Schedule.DueAfter.Before(baseline) {
		t.Errorf("due %s -> %s, want an earlier date than the baseline %s", rep.Schedule.DueBefore, rep.Schedule.DueAfter, baseline)
	}
	if got := e.teachBacks(); got != eventsBefore+1 {
		t.Errorf("teach_back events %d -> %d, want one more", eventsBefore, got)
	}

	// The weekly budget for the subject moved: projected recall at the exam fell, so more hours are asked for.
	if after := e.nextWeekHours(); after == hoursBefore {
		t.Errorf("next week hours stayed %.2f after a wrong explanation", after)
	}

	// A new gap card, shown on the review screen once the concept is due.
	if n := e.count(`SELECT count(*) FROM cards WHERE concept_id = $1`, fxConcept); n == 0 {
		t.Error("no gap card was made")
	}
	e.clock.Advance(72 * time.Hour)
	var due []domain.DueItem
	e.call("GET", "/v1/reviews/due", "", 200, &due)
	var card *domain.Card
	for _, it := range due {
		if it.Card != nil && it.ConceptID == fxConcept {
			card = it.Card
			if it.Reason == "" {
				t.Errorf("due item has no reason: %+v", it)
			}
		}
	}
	if card == nil {
		t.Fatalf("no gap card in the due list: %+v", due)
	}
	var cev domain.ReviewEvent
	e.call("POST", "/v1/cards/"+card.ID+"/review", `{"grade":"good"}`, 200, &cev)
	if cev.EventType != domain.EventCardReview || cev.CardID == nil || *cev.CardID != card.ID {
		t.Errorf("card review event = %+v", cev)
	}

	// Finish again: same report, no second event.
	afterOne := e.count(`SELECT count(*) FROM review_events WHERE event_type = 'teach_back' AND concept_id = $1`, fxConcept)
	var again domain.Report
	e.call("POST", "/v1/teach/sessions/"+ts.ID+"/finish", "", 200, &again)
	if again.Schedule == nil || again.Schedule.ID != rep.Schedule.ID {
		t.Errorf("second finish returned a different event: %+v", again.Schedule)
	}
	if got := e.teachBacks(); got != afterOne {
		t.Errorf("repeat finish wrote %d teach_back events, want %d", got, afterOne)
	}

	// The report endpoint returns the same report.
	var got domain.Report
	e.call("GET", "/v1/teach/sessions/"+ts.ID+"/report", "", 200, &got)
	if got.Schedule == nil || got.Schedule.ID != rep.Schedule.ID || got.Status != domain.ReportChecked {
		t.Errorf("report endpoint = %+v", got)
	}
}

func ptr(s string) *string { return &s }

// checkerReply is a scripted checker answer for plantedMistake, citing the fixture source.
func checkerReply() string {
	b, _ := json.Marshal(map[string]any{"results": []domain.ClaimResult{
		{ClaimID: ptr(fxClaim1), StudentText: "The chance that X finishes before Y is b/(a+b).", Label: domain.LabelContradicted,
			Citation:   &domain.Citation{SourceID: fxSource, Quote: "Then the chance that X finishes before Y is a / (a + b).", Locator: "p.1"},
			Correction: ptr("The chance is a/(a+b): the faster rate goes on top, not b."), ProbeHint: ptr("which rate belongs in the numerator")},
		{ClaimID: ptr(fxClaim2), StudentText: "The smaller of the two times is exponential with rate a+b.", Label: domain.LabelSupported},
	}})
	return string(b)
}

func TestFullLoopFakeModel(t *testing.T) {
	student := &fakes.LLM{Reply: "Why does the rate b go on top?"}
	e := newEnv(t, student, &fakes.LLM{JSONReply: checkerReply()}, "")
	runLoop(t, e)

	// The AI student never saw source text.
	for _, r := range student.Requests {
		all := r.System
		for _, m := range r.Messages {
			all += m.Content
		}
		if strings.Contains(all, "condition on X = t") || strings.Contains(all, "a / (a + b)") {
			t.Errorf("student request contains source text: %q", all)
		}
	}
}

type failingLLM struct{}

func (failingLLM) Stream(context.Context, domain.LLMRequest) (<-chan domain.Delta, error) {
	return nil, errors.New("down")
}
func (failingLLM) JSON(context.Context, domain.LLMRequest, any) error {
	return errors.New("checker is down")
}

// With the checker down, Finish still answers, the report is unchecked and no
// teach_back event is written, so no review date moves.
func TestCheckerDownGivesUncheckedReport(t *testing.T) {
	e := newEnv(t, &fakes.LLM{Reply: "Why does that happen?"}, failingLLM{}, "")
	before := e.teachBacks()
	var ts domain.TeachSession
	e.call("POST", "/v1/teach/sessions", fmt.Sprintf(`{"concept_id":%q}`, fxConcept), 201, &ts)
	e.runTurn(ts.ID, plantedMistake)
	var rep domain.Report
	e.call("POST", "/v1/teach/sessions/"+ts.ID+"/finish", "", 200, &rep)
	if rep.Status != domain.ReportUnchecked || rep.Schedule != nil {
		t.Errorf("status %q, schedule set: %v; want unchecked with no schedule", rep.Status, rep.Schedule != nil)
	}
	if got := e.teachBacks(); got != before {
		t.Errorf("teach_back events %d -> %d, want none written", before, got)
	}
	if n := e.count(`SELECT count(*) FROM cards WHERE concept_id = $1`, fxConcept); n != 0 {
		t.Errorf("%d cards made from an unchecked session", n)
	}
	var got domain.Report
	e.call("GET", "/v1/teach/sessions/"+ts.ID+"/report", "", 200, &got)
	if got.Status != domain.ReportUnchecked {
		t.Errorf("report endpoint status %q, want unchecked", got.Status)
	}
}
