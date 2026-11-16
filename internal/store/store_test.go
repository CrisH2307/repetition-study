package store_test

import (
	"context"
	"fmt"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/CrisH2307/repetition-study/internal/domain"
	"github.com/CrisH2307/repetition-study/internal/store"
	"github.com/CrisH2307/repetition-study/internal/teach"
	"github.com/CrisH2307/repetition-study/migrations"
)

const (
	subA, subB             = "00000000-0000-0000-0000-0000000000a1", "00000000-0000-0000-0000-0000000000a2"
	srcCourse, srcAllow    = "00000000-0000-0000-0000-0000000000b1", "00000000-0000-0000-0000-0000000000b2"
	srcOther               = "00000000-0000-0000-0000-0000000000b3"
	conc1, conc2, concB    = "00000000-0000-0000-0000-000000000101", "00000000-0000-0000-0000-000000000102", "00000000-0000-0000-0000-000000000201"
	claimReq, claimOpt     = "00000000-0000-0000-0000-000000020101", "00000000-0000-0000-0000-000000020102"
	claimAllow, claimOther = "00000000-0000-0000-0000-000000020103", "00000000-0000-0000-0000-000000020201"
)

func pack() domain.Pack {
	rev := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	page := 2
	return domain.Pack{
		Subject: domain.Subject{ID: subA, Name: "A", WeeklyHours: 4, Level: domain.LevelIntermediate, Weight: 1},
		Sources: []domain.Source{
			{ID: srcCourse, Kind: domain.SourceCourse, Title: "course", Text: "alpha beta", ReviewedBy: "cris", ReviewedAt: &rev},
			{ID: srcAllow, Kind: domain.SourceAllowlist, Title: "outside", Text: "gamma"},
			{ID: srcOther, Kind: domain.SourceCourse, Title: "other subject", Text: "delta"},
		},
		Blocks: []domain.SourceBlock{{ID: "00000000-0000-0000-0000-000000030001", SourceID: srcCourse, Page: &page,
			Ordinal: 1, Kind: domain.BlockText, TextMD: "alpha  beta", TextNorm: "alpha beta"}},
		Concepts: []domain.Concept{
			{ID: conc1, SubjectID: subA, Name: "one", Claims: []domain.Claim{
				{ID: claimReq, ConceptID: conc1, Text: "req", SourceID: srcCourse, Locator: "p.1", Required: true},
				{ID: claimOpt, ConceptID: conc1, Text: "opt", SourceID: srcCourse, Required: false}}},
			{ID: conc2, SubjectID: subA, Name: "two", Claims: []domain.Claim{
				{ID: claimAllow, ConceptID: conc2, Text: "x", SourceID: srcAllow, Required: true}}},
		},
	}
}

// otherSubject is loaded as a second pack so srcOther belongs to another subject.
func otherSubject() domain.Pack {
	return domain.Pack{
		Subject: domain.Subject{ID: subB, Name: "B", WeeklyHours: 1, Level: domain.LevelBeginner, Weight: 1},
		Sources: []domain.Source{{ID: srcOther, Kind: domain.SourceCourse, Title: "other subject", Text: "delta"}},
		Concepts: []domain.Concept{{ID: concB, SubjectID: subB, Name: "b", Claims: []domain.Claim{
			{ID: claimOther, ConceptID: concB, Text: "y", SourceID: srcOther, Required: true}}}},
	}
}

// conformance runs against any domain.TeachStore; concept conc1 must exist.
func conformance(t *testing.T, s domain.TeachStore) {
	ctx := context.Background()
	c, srcs, err := s.LoadConcept(ctx, conc1)
	if err != nil || c.ID != conc1 || len(c.Claims) == 0 {
		t.Fatalf("LoadConcept: %+v %v", c, err)
	}
	found := false
	for _, x := range srcs {
		found = found || x.ID == srcCourse
	}
	if !found {
		t.Fatalf("course source missing from %v", srcs)
	}
	if _, _, err := s.LoadConcept(ctx, "00000000-0000-0000-0000-00000000ffff"); err == nil {
		t.Fatal("unknown concept must error")
	}

	sess, err := s.StartSession(ctx, conc1)
	if err != nil || sess.ID == "" || sess.ConceptID != conc1 {
		t.Fatalf("StartSession: %+v %v", sess, err)
	}
	ids := map[string]bool{}
	for n := 1; n <= 2; n++ {
		tc, err := s.SaveTurn(ctx, domain.TaughtClaim{SessionID: sess.ID, ConceptID: conc1, StudentText: fmt.Sprint("said ", n), TurnNo: n}, "tr", `\x`)
		if err != nil || tc.ID == "" || ids[tc.ID] || tc.StudentText != fmt.Sprint("said ", n) || tc.TurnNo != n || tc.SessionID != sess.ID {
			t.Fatalf("SaveTurn %d: %+v %v", n, tc, err)
		}
		ids[tc.ID] = true
		if err := s.SetAssistantText(ctx, sess.ID, n, "why?"); err != nil {
			t.Fatal(err)
		}
	}
	req, corr, q := claimReq, "fix", "alpha beta"
	first := []domain.ClaimResult{{ClaimID: &req, StudentText: "said 1", Label: domain.LabelContradicted,
		Citation: &domain.Citation{SourceID: srcCourse, Quote: q, Locator: "p.1"}, Correction: &corr}}
	second := []domain.ClaimResult{{StudentText: "extra", Label: domain.LabelAbsent}, {ClaimID: &req, StudentText: "said 1", Label: domain.LabelSupported}}
	for _, r := range [][]domain.ClaimResult{first, second} { // second replaces first
		if err := s.SaveResults(ctx, sess.ID, r); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 2; i++ { // Finish may be retried
		if err := s.FinishSession(ctx, sess.ID); err != nil {
			t.Fatalf("FinishSession #%d: %v", i, err)
		}
	}
}

func TestMemStoreConformance(t *testing.T) {
	p := pack()
	var claims []domain.Claim
	claims = append(claims, p.Concepts[0].Claims...)
	c := p.Concepts[0]
	c.Claims = claims
	conformance(t, teach.NewMemStore(p.Sources, c))
}

// ---- Postgres ----

// openDB makes a throwaway schema with every migration applied and returns a
// connect func. Each call to connect is a fresh connection ("restart").
func openDB(t *testing.T) (connect func() *pgx.Conn) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set: skipping Postgres store tests (run `make up migrate`, then set it)")
	}
	ctx := context.Background()
	admin, err := pgx.Connect(ctx, url)
	if err != nil {
		t.Skipf("cannot reach Postgres (%v): skipping", err)
	}
	schema := fmt.Sprintf("store_test_%d", time.Now().UnixNano())
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		admin.Exec(context.Background(), "DROP SCHEMA "+schema+" CASCADE")
		admin.Close(context.Background())
	})
	cfg, err := pgx.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	cfg.RuntimeParams["search_path"] = schema
	connect = func() *pgx.Conn {
		c, err := pgx.ConnectConfig(ctx, cfg)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { c.Close(context.Background()) })
		return c
	}
	conn := connect()
	files, _ := migrations.FS.ReadDir(".")
	var names []string
	for _, f := range files {
		if len(f.Name()) > 4 && f.Name()[len(f.Name())-4:] == ".sql" {
			names = append(names, f.Name())
		}
	}
	sort.Strings(names)
	for _, n := range names {
		sql, _ := migrations.FS.ReadFile(n)
		if _, err := conn.Exec(ctx, string(sql)); err != nil {
			t.Fatalf("%s: %v", n, err)
		}
	}
	return connect
}

func loaded(t *testing.T) (func() *pgx.Conn, *store.Store) {
	connect := openDB(t)
	s := store.New(connect())
	for _, p := range []domain.Pack{pack(), otherSubject()} {
		if err := s.LoadPack(context.Background(), p); err != nil {
			t.Fatal(err)
		}
	}
	return connect, s
}

func count(t *testing.T, c *pgx.Conn, sql string, args ...any) (n int) {
	t.Helper()
	if err := c.QueryRow(context.Background(), sql, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return
}

func TestPostgresConformance(t *testing.T) {
	_, s := loaded(t)
	conformance(t, s)
}

func TestLoadConceptApprovedSourcesOnly(t *testing.T) {
	_, s := loaded(t)
	c, srcs, err := s.LoadConcept(context.Background(), conc1)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Claims) != 1 || c.Claims[0].ID != claimReq {
		t.Fatalf("want only the required claim, got %+v", c.Claims)
	}
	// srcAllow is cited by a claim of this subject but is allowlist; srcOther is another subject's.
	if len(srcs) != 1 || srcs[0].ID != srcCourse || srcs[0].Text != "alpha beta" || srcs[0].ReviewedBy != "cris" {
		t.Fatalf("sources = %+v", srcs)
	}
}

func TestSessionSurvivesRestartAndFinishWritesLabels(t *testing.T) {
	connect, s := loaded(t)
	ctx := context.Background()
	sess, err := s.StartSession(ctx, conc1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.SaveTurn(ctx, domain.TaughtClaim{SessionID: sess.ID, StudentText: "wrong idea", TurnNo: 1}, "tr", ""); err != nil {
		t.Fatal(err)
	}

	// "Restart": a brand new connection and a new Store. Nothing is held in memory.
	c2 := connect()
	s2 := store.New(c2)
	req, corr := claimReq, "no, see source"
	res := []domain.ClaimResult{{ClaimID: &req, StudentText: "wrong idea", Label: domain.LabelContradicted,
		Citation: &domain.Citation{SourceID: srcCourse, Quote: "alpha beta", Locator: "p.1"}, Correction: &corr}}
	if err := s2.SetAssistantText(ctx, sess.ID, 1, "why is that?"); err != nil {
		t.Fatal(err)
	}
	if err := s2.SaveResults(ctx, sess.ID, res); err != nil {
		t.Fatal(err)
	}
	if err := s2.FinishSession(ctx, sess.ID); err != nil {
		t.Fatal(err)
	}
	if n := count(t, c2, `SELECT count(*) FROM claim_labels WHERE session_id = $1 AND label = 'contradicted' AND correction = $2 AND citation_quote = 'alpha beta'`, sess.ID, corr); n != 1 {
		t.Fatalf("claim_labels rows = %d, want 1", n)
	}
	if n := count(t, c2, `SELECT count(*) FROM teach_sessions WHERE id = $1 AND status = 'finished' AND finished_at IS NOT NULL`, sess.ID); n != 1 {
		t.Fatal("session not finished")
	}
	if n := count(t, c2, `SELECT count(*) FROM teach_turns WHERE session_id = $1 AND assistant_text = 'why is that?'`, sess.ID); n != 1 {
		t.Fatal("assistant text not saved")
	}
	if n := count(t, c2, `SELECT count(*) FROM taught_claims WHERE session_id = $1 AND concept_id = $2 AND student_text = 'wrong idea'`, sess.ID, conc1); n != 1 {
		t.Fatal("taught claim missing or concept not filled from session")
	}
}

func TestSaveResultsReplacesAndUnknownRowsError(t *testing.T) {
	connect, s := loaded(t)
	ctx := context.Background()
	sess, _ := s.StartSession(ctx, conc1)
	for i := 0; i < 3; i++ {
		if err := s.SaveResults(ctx, sess.ID, []domain.ClaimResult{{StudentText: "a", Label: domain.LabelAbsent}, {StudentText: "b", Label: domain.LabelPartial}}); err != nil {
			t.Fatal(err)
		}
	}
	if n := count(t, connect(), `SELECT count(*) FROM claim_labels WHERE session_id = $1`, sess.ID); n != 2 {
		t.Fatalf("rows = %d, want 2 (replace, not append)", n)
	}
	// A correction without a citation is rejected by the table, and the old labels stay.
	bad := "x"
	if err := s.SaveResults(ctx, sess.ID, []domain.ClaimResult{{StudentText: "a", Label: domain.LabelPartial, Correction: &bad}}); err == nil {
		t.Fatal("correction without citation must fail")
	}
	if n := count(t, connect(), `SELECT count(*) FROM claim_labels WHERE session_id = $1`, sess.ID); n != 2 {
		t.Fatalf("failed save must roll back, rows = %d", n)
	}
	missing := "00000000-0000-0000-0000-00000000ffff"
	if err := s.FinishSession(ctx, missing); err == nil {
		t.Fatal("finishing an unknown session must error")
	}
	if err := s.SetAssistantText(ctx, sess.ID, 9, "x"); err == nil {
		t.Fatal("unknown turn must error")
	}
}

func TestLoadPackIsIdempotent(t *testing.T) {
	connect, s := loaded(t)
	ctx := context.Background()
	p := pack()
	cid := conc1
	p.History = []domain.ReviewEvent{{ID: "00000000-0000-0000-0000-000000040001", EventType: domain.EventTeachBack, ConceptID: cid,
		DueBefore: time.Now(), DueAfter: time.Now(), CreatedAt: time.Now()}}
	for i := 0; i < 2; i++ {
		if err := s.LoadPack(ctx, p); err != nil {
			t.Fatal(err)
		}
	}
	c := connect()
	for table, want := range map[string]int{"subjects": 2, "sources": 3, "source_blocks": 1, "concepts": 3, "claims": 4, "review_events": 1} {
		if n := count(t, c, "SELECT count(*) FROM "+table); n != want {
			t.Errorf("%s = %d, want %d", table, n, want)
		}
	}
	// A reloaded pack updates edited rows.
	p.Concepts[0].Claims[0].Text = "edited"
	if err := s.LoadPack(ctx, p); err != nil {
		t.Fatal(err)
	}
	if n := count(t, c, `SELECT count(*) FROM claims WHERE text = 'edited'`); n != 1 {
		t.Fatal("claim edit not applied")
	}
	// A pack without ids is refused rather than half loaded.
	if err := s.LoadPack(ctx, domain.Pack{}); err == nil {
		t.Fatal("empty pack must error")
	}
}

func TestLoadPackSetsSourceSubject(t *testing.T) {
	connect, s := loaded(t)
	ctx := context.Background()
	sub := pack().Subject.ID
	srcs, err := s.CourseSources(ctx, domain.DemoUserID, sub)
	if err != nil || len(srcs) != 1 || srcs[0].ID != srcCourse {
		t.Fatalf("course sources = %v, %v", srcs, err)
	}
	// A reload fills a null subject_id left by an older load.
	c := connect()
	if _, err := c.Exec(ctx, `UPDATE sources SET subject_id = NULL`); err != nil {
		t.Fatal(err)
	}
	if err := s.LoadPack(ctx, pack()); err != nil {
		t.Fatal(err)
	}
	if srcs, err = s.CourseSources(ctx, domain.DemoUserID, sub); err != nil || len(srcs) == 0 || srcs[0].ID != srcCourse {
		t.Fatalf("after reload = %v, %v", srcs, err)
	}
}
