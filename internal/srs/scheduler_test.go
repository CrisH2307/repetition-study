package srs

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/CrisH2307/repetition-study/internal/domain"
	"github.com/CrisH2307/repetition-study/internal/fakes"
	"github.com/CrisH2307/repetition-study/migrations"
)

// Tests need a running Postgres (DATABASE_URL). Each test creates a throwaway
// database on that server, applies every migration to it and drops it at the
// end, so the dev database gains no rows. They skip without DATABASE_URL.
type env struct {
	t     *testing.T
	ctx   context.Context
	conn  *pgx.Conn
	clock *fakes.Clock
	s     *Scheduler
}

func newEnv(t *testing.T) *env {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set")
	}
	ctx := context.Background()
	conn := throwawayDB(t, ctx, url)
	if _, err := conn.Exec(ctx, `INSERT INTO users (id, name) VALUES ($1, 'demo') ON CONFLICT DO NOTHING`, domain.DemoUserID); err != nil {
		t.Fatal(err)
	}
	clock := fakes.NewClock()
	return &env{t, ctx, conn, clock, New(conn, clock)}
}

// throwawayDB makes an empty database on the server in url, migrates it and
// drops it when the test ends.
func throwawayDB(t *testing.T, ctx context.Context, url string) *pgx.Conn {
	cfg, err := pgx.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	admin, err := pgx.ConnectConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	name := "srs_test_" + strings.ReplaceAll(fmt.Sprint(time.Now().UnixNano()), "-", "")
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+name); err != nil {
		admin.Close(ctx)
		t.Fatal(err)
	}
	cfg.Database = name
	conn, err := pgx.ConnectConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		conn.Close(ctx)
		if _, err := admin.Exec(ctx, "DROP DATABASE "+name+" WITH (FORCE)"); err != nil {
			t.Errorf("drop %s: %v", name, err)
		}
		admin.Close(ctx)
	})
	files, err := migrations.FS.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files { // ReadDir is sorted by name
		if !strings.HasSuffix(f.Name(), ".sql") {
			continue
		}
		b, _ := migrations.FS.ReadFile(f.Name())
		if _, err := conn.Exec(ctx, string(b)); err != nil {
			t.Fatalf("%s: %v", f.Name(), err)
		}
	}
	return conn
}

func (e *env) q(sql string, args ...any) string {
	var id string
	if err := e.conn.QueryRow(e.ctx, sql, args...).Scan(&id); err != nil {
		e.t.Fatal(err)
	}
	return id
}

// concept makes a concept with a required claim, a second required claim and an optional one.
func (e *env) concept() (id string, required, required2, optional string) {
	subj := e.q(`INSERT INTO subjects (name, weekly_hours, level) VALUES ('srs test', 1, 'beginner') RETURNING id`)
	src := e.q(`INSERT INTO sources (kind, title, text) VALUES ('course', 'srs test', 'x') RETURNING id`)
	id = e.q(`INSERT INTO concepts (subject_id, name) VALUES ($1, 'concept ' || gen_random_uuid()) RETURNING id`, subj)
	claim := func(req bool) string {
		return e.q(`INSERT INTO claims (concept_id, text, source_id, required) VALUES ($1, 'claim text', $2, $3) RETURNING id`, id, src, req)
	}
	return id, claim(true), claim(true), claim(false)
}

func (e *env) eventCount(id string) (n int) {
	if err := e.conn.QueryRow(e.ctx, `SELECT count(*) FROM review_events WHERE concept_id = $1`, id).Scan(&n); err != nil {
		e.t.Fatal(err)
	}
	return n
}

func (e *env) cardCount(id string) (n int) {
	if err := e.conn.QueryRow(e.ctx, `SELECT count(*) FROM cards WHERE concept_id = $1`, id).Scan(&n); err != nil {
		e.t.Fatal(err)
	}
	return n
}

func lbl(claim string, l domain.ClaimLabel) domain.ClaimResult {
	return domain.ClaimResult{ClaimID: &claim, StudentText: "student said this", Label: l}
}

func TestApplyTeachBackGradeMappings(t *testing.T) {
	e := newEnv(t)
	cases := []struct {
		name  string
		label domain.ClaimLabel
		want  domain.Grade
	}{
		{"contradicted", domain.LabelContradicted, domain.GradeAgain},
		{"absent", domain.LabelAbsent, domain.GradeAgain},
		{"partial", domain.LabelPartial, domain.GradeHard},
		{"supported", domain.LabelSupported, domain.GradeGood},
	}
	for _, c := range cases {
		id, r1, r2, opt := e.concept()
		// the optional claim is always contradicted and must not change the grade
		ev, err := e.s.ApplyTeachBack(e.ctx, id, []domain.ClaimResult{lbl(r1, domain.LabelSupported), lbl(r2, c.label), lbl(opt, domain.LabelContradicted)})
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if ev.EventType != domain.EventTeachBack || ev.Grade == nil || *ev.Grade != c.want {
			t.Errorf("%s: got %v %v, want teach_back %s", c.name, ev.EventType, ev.Grade, c.want)
		}
		if !ev.CreatedAt.Equal(fakes.Epoch) {
			t.Errorf("%s: event time %v is not the fixed clock", c.name, ev.CreatedAt)
		}
	}
}

func TestContradictedTeachBackPullsReviewEarlier(t *testing.T) {
	e := newEnv(t)
	id, r1, _, _ := e.concept()
	ev, err := e.s.ApplyTeachBack(e.ctx, id, []domain.ClaimResult{lbl(r1, domain.LabelSupported)})
	for i := 0; err == nil && ev.DueAfter.Sub(e.clock.Now()) < 7*24*time.Hour && i < 20; i++ {
		e.clock.Advance(ev.DueAfter.Sub(e.clock.Now()))
		ev, err = e.s.ApplyTeachBack(e.ctx, id, []domain.ClaimResult{lbl(r1, domain.LabelSupported)})
	}
	if err != nil {
		t.Fatal(err)
	}
	if ev.DueAfter.Sub(e.clock.Now()) < 7*24*time.Hour {
		t.Fatalf("setup: wanted a concept due in 7+ days, got %v", ev.DueAfter.Sub(e.clock.Now()))
	}
	bad, err := e.s.ApplyTeachBack(e.ctx, id, []domain.ClaimResult{lbl(r1, domain.LabelContradicted)})
	if err != nil {
		t.Fatal(err)
	}
	if !bad.DueBefore.Equal(ev.DueAfter) || !bad.DueAfter.Before(bad.DueBefore) {
		t.Fatalf("due before %v after %v, want after earlier than %v", bad.DueBefore, bad.DueAfter, ev.DueAfter)
	}
	if bad.StabilityAfter >= bad.StabilityBefore {
		t.Errorf("stability %v -> %v should drop", bad.StabilityBefore, bad.StabilityAfter)
	}
}

func TestOverrideLeavesMemoryAlone(t *testing.T) {
	e := newEnv(t)
	id, r1, _, _ := e.concept()
	if _, err := e.s.ApplyTeachBack(e.ctx, id, []domain.ClaimResult{lbl(r1, domain.LabelSupported)}); err != nil {
		t.Fatal(err)
	}
	type row struct {
		stab, diff          float64
		state, reps, lapses int
	}
	read := func() (r row) {
		if err := e.conn.QueryRow(e.ctx, `SELECT stability, difficulty, state, reps, lapses FROM concept_schedules WHERE concept_id=$1`, id).
			Scan(&r.stab, &r.diff, &r.state, &r.reps, &r.lapses); err != nil {
			t.Fatal(err)
		}
		return r
	}
	was := read()
	to := fakes.Epoch.Add(30 * 24 * time.Hour)
	ev, err := e.s.Override(e.ctx, id, to, "exam moved")
	if err != nil {
		t.Fatal(err)
	}
	if read() != was || ev.StabilityBefore != ev.StabilityAfter {
		t.Errorf("memory state changed by override: %+v -> %+v", was, read())
	}
	if ev.EventType != domain.EventManualOverride || !ev.DueAfter.Equal(to) || !strings.Contains(string(ev.Payload), "exam moved") {
		t.Errorf("bad override event: %+v payload %s", ev, ev.Payload)
	}
	if _, err := e.s.Override(e.ctx, id, to, ""); err == nil {
		t.Error("override without a reason should fail")
	}
}

func TestRetriedEventWritesOnce(t *testing.T) {
	e := newEnv(t)
	id, r1, _, _ := e.concept()
	ctx := WithEventID(e.ctx, e.q(`SELECT gen_random_uuid()`))
	bad := []domain.ClaimResult{func() domain.ClaimResult {
		r := lbl(r1, domain.LabelContradicted)
		c := "correct version"
		r.Correction = &c
		r.Citation = &domain.Citation{SourceID: "s", Quote: "the quote", Locator: "p. 4"}
		return r
	}()}
	first, err := e.s.ApplyTeachBack(ctx, id, bad)
	if err != nil {
		t.Fatal(err)
	}
	e.clock.Advance(time.Hour) // a retry arrives later; it must not move anything
	again, err := e.s.ApplyTeachBack(ctx, id, bad)
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != again.ID || !first.DueAfter.Equal(again.DueAfter) {
		t.Errorf("retry returned a different event: %+v vs %+v", first, again)
	}
	if n := e.eventCount(id); n != 1 {
		t.Errorf("%d events, want 1", n)
	}
	var reps int
	e.conn.QueryRow(e.ctx, `SELECT reps FROM concept_schedules WHERE concept_id=$1`, id).Scan(&reps)
	if reps != 1 {
		t.Errorf("reps %d, want 1", reps)
	}
}

func TestGapCards(t *testing.T) {
	e := newEnv(t)
	id, r1, r2, _ := e.concept()
	c := "the right answer"
	bad := lbl(r1, domain.LabelContradicted)
	bad.Correction = &c
	bad.Citation = &domain.Citation{SourceID: "s", Quote: "exact quote", Locator: "p. 4"}
	results := []domain.ClaimResult{bad, lbl(r2, domain.LabelAbsent)}
	for i := 0; i < 2; i++ { // second teach-back must not duplicate open gap cards
		if _, err := e.s.ApplyTeachBack(e.ctx, id, results); err != nil {
			t.Fatal(err)
		}
		if n := e.cardCount(id); n != 2 {
			t.Fatalf("round %d: %d cards, want 2", i, n)
		}
	}
	var back string
	e.conn.QueryRow(e.ctx, `SELECT back FROM cards WHERE concept_id=$1 AND claim_id=$2`, id, r1).Scan(&back)
	if !strings.Contains(back, "the right answer") || !strings.Contains(back, `"exact quote" (p. 4)`) {
		t.Errorf("card back lacks correction or citation: %q", back)
	}
	// a supported result makes no card
	id2, s1, _, _ := e.concept()
	if _, err := e.s.ApplyTeachBack(e.ctx, id2, []domain.ClaimResult{lbl(s1, domain.LabelSupported)}); err != nil {
		t.Fatal(err)
	}
	if n := e.cardCount(id2); n != 0 {
		t.Errorf("supported result made %d cards", n)
	}
}

func TestDueReasonsAndInterleave(t *testing.T) {
	e := newEnv(t)
	a, a1, a2, _ := e.concept()
	b, b1, b2, _ := e.concept()
	for _, c := range [][3]string{{a, a1, a2}, {b, b1, b2}} {
		if _, err := e.s.ApplyTeachBack(e.ctx, c[0], []domain.ClaimResult{lbl(c[1], domain.LabelContradicted), lbl(c[2], domain.LabelAbsent)}); err != nil {
			t.Fatal(err)
		}
	}
	e.clock.Advance(48 * time.Hour)
	got, err := e.s.DueWithReasons(e.ctx, e.clock.Now())
	if err != nil {
		t.Fatal(err)
	}
	var mine []ReasonedDue
	for _, d := range got {
		if d.ConceptID == a || d.ConceptID == b {
			mine = append(mine, d)
		}
	}
	if len(mine) != 6 { // 2 concepts + 2 cards each
		t.Fatalf("%d items, want 6", len(mine))
	}
	for i, d := range mine {
		if d.Reason == "" || (d.CardID != nil && d.Card == nil) {
			t.Errorf("item %d: reason %q card %v", i, d.Reason, d.Card)
		}
		if i > 0 && d.ConceptID == mine[i-1].ConceptID && i < 4 {
			t.Errorf("same concept twice in a row at %d with another concept due", i)
		}
	}
	if !strings.Contains(mine[0].Reason, "teach-back found gaps (again)") {
		t.Errorf("reason %q", mine[0].Reason)
	}
	plain, err := e.s.Due(e.ctx, e.clock.Now())
	if err != nil || len(plain) != len(got) {
		t.Errorf("Due returned %d, DueWithReasons %d (%v)", len(plain), len(got), err)
	}
}

func TestReviewCardMovesConceptSchedule(t *testing.T) {
	e := newEnv(t)
	id, r1, _, _ := e.concept()
	if _, err := e.s.ApplyTeachBack(e.ctx, id, []domain.ClaimResult{lbl(r1, domain.LabelPartial)}); err != nil {
		t.Fatal(err)
	}
	var card string
	e.conn.QueryRow(e.ctx, `SELECT id FROM cards WHERE concept_id=$1`, id).Scan(&card)
	ev, err := e.s.ReviewCard(e.ctx, card, domain.GradeEasy)
	if err != nil {
		t.Fatal(err)
	}
	if ev.EventType != domain.EventCardReview || ev.CardID == nil || *ev.CardID != card || *ev.Grade != domain.GradeEasy {
		t.Errorf("bad event %+v", ev)
	}
}

// Finish retried after a restart: a brand new Scheduler (and clock) applies the
// same session again and must get the first event back.
func TestTeachBackSessionIsIdempotent(t *testing.T) {
	e := newEnv(t)
	id, r1, _, _ := e.concept()
	sess := e.q(`INSERT INTO teach_sessions (concept_id) VALUES ($1) RETURNING id`, id)
	res := []domain.ClaimResult{lbl(r1, domain.LabelContradicted)}
	first, err := e.s.ApplyTeachBackSession(e.ctx, sess, id, res)
	if err != nil {
		t.Fatal(err)
	}
	if first.SessionID == nil || *first.SessionID != sess {
		t.Errorf("event session id %v, want %s", first.SessionID, sess)
	}
	restarted := New(e.conn, fakes.NewClock())
	restarted.clock.(*fakes.Clock).Advance(48 * time.Hour)
	again, err := restarted.ApplyTeachBackSession(e.ctx, sess, id, res)
	if err != nil {
		t.Fatal(err)
	}
	if again.ID != first.ID || !again.DueAfter.Equal(first.DueAfter) {
		t.Errorf("retry returned another event: %+v vs %+v", again, first)
	}
	if n := e.eventCount(id); n != 1 {
		t.Errorf("%d events, want 1", n)
	}
	if n := e.cardCount(id); n != 1 {
		t.Errorf("%d cards, want 1", n)
	}
	// A different session on the same concept is a new teach-back.
	sess2 := e.q(`INSERT INTO teach_sessions (concept_id) VALUES ($1) RETURNING id`, id)
	if _, err := e.s.ApplyTeachBackSession(e.ctx, sess2, id, res); err != nil {
		t.Fatal(err)
	}
	if n := e.eventCount(id); n != 2 {
		t.Errorf("%d events, want 2", n)
	}
	if _, err := e.s.ApplyTeachBackSession(e.ctx, "", id, res); err == nil {
		t.Error("empty session id should fail")
	}
}

func TestTeachBackSessionConcurrentRetries(t *testing.T) {
	e := newEnv(t)
	id, r1, _, _ := e.concept()
	sess := e.q(`INSERT INTO teach_sessions (concept_id) VALUES ($1) RETURNING id`, id)
	url := os.Getenv("DATABASE_URL")
	cfg, _ := pgx.ParseConfig(url)
	cfg.Database = e.conn.Config().Database
	ids := make(chan string, 4)
	for i := 0; i < 4; i++ {
		go func() {
			c, err := pgx.ConnectConfig(e.ctx, cfg)
			if err != nil {
				ids <- "err " + err.Error()
				return
			}
			defer c.Close(e.ctx)
			ev, err := New(c, e.clock).ApplyTeachBackSession(e.ctx, sess, id, []domain.ClaimResult{lbl(r1, domain.LabelContradicted)})
			if err != nil {
				ids <- "err " + err.Error()
				return
			}
			ids <- ev.ID
		}()
	}
	seen := map[string]bool{}
	for i := 0; i < 4; i++ {
		seen[<-ids] = true
	}
	if len(seen) != 1 || e.eventCount(id) != 1 {
		t.Errorf("want one event, got ids %v and %d rows", seen, e.eventCount(id))
	}
}

func TestReviewEventsStayAppendOnly(t *testing.T) {
	e := newEnv(t)
	id, r1, _, _ := e.concept()
	if _, err := e.s.ApplyTeachBack(e.ctx, id, []domain.ClaimResult{lbl(r1, domain.LabelSupported)}); err != nil {
		t.Fatal(err)
	}
	if _, err := e.conn.Exec(e.ctx, `UPDATE review_events SET grade = 'easy' WHERE concept_id = $1`, id); err == nil {
		t.Error("update on review_events should be rejected")
	}
	if _, err := e.conn.Exec(e.ctx, `DELETE FROM review_events WHERE concept_id = $1`, id); err == nil {
		t.Error("delete on review_events should be rejected")
	}
}
