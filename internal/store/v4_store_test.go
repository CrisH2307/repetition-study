package store_test

import (
	"sync"
	"testing"
	"time"

	"github.com/CrisH2307/repetition-study/internal/domain"
	"github.com/CrisH2307/repetition-study/internal/store"
)

func TestFeedbackAndConsent(t *testing.T) {
	s := store.New(openDB(t)())
	a, b := newUser(t, s, "A"), newUser(t, s, "B")
	wk := time.Date(2026, 11, 16, 0, 0, 0, 0, time.UTC)

	_, err := s.GetFeedback(bg, a.ID, wk)
	isNF(t, "no answer yet", err)
	got, err := s.SaveFeedback(bg, a.ID, domain.FeedbackAnswer{WeekStart: wk, Question: "q1", Answer: "first"})
	mustOK(t, err)
	if !got.WeekStart.Equal(wk) || got.Answer != "first" || got.CreatedAt.IsZero() {
		t.Fatalf("save: %+v", got)
	}
	got, err = s.SaveFeedback(bg, a.ID, domain.FeedbackAnswer{WeekStart: wk, Question: "q1", Answer: "second"})
	mustOK(t, err)
	if got.Answer != "second" {
		t.Fatalf("replace: %+v", got)
	}
	got, err = s.GetFeedback(bg, a.ID, wk)
	if err != nil || got.Answer != "second" {
		t.Fatalf("get: %+v %v", got, err)
	}
	_, err = s.GetFeedback(bg, b.ID, wk)
	isNF(t, "other user's answer", err)
	_, err = s.GetFeedback(bg, a.ID, wk.AddDate(0, 0, 7))
	isNF(t, "other week", err)
	_, err = s.SaveFeedback(bg, "not-a-uuid", domain.FeedbackAnswer{WeekStart: wk, Question: "q", Answer: "a"})
	isNF(t, "bad user id", err)

	t1, t2 := time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC), time.Date(2026, 11, 5, 9, 0, 0, 0, time.UTC)
	c, err := s.SaveConsent(bg, a.ID, "v1", t1)
	mustOK(t, err)
	if c.TextVersion != "v1" || !c.AcceptedAt.Equal(t1) {
		t.Fatalf("consent: %+v", c)
	}
	c, err = s.SaveConsent(bg, a.ID, "v1", t2)
	mustOK(t, err)
	if !c.AcceptedAt.Equal(t1) {
		t.Fatalf("first accepted_at must stay: %+v", c)
	}
	_, err = s.SaveConsent(bg, a.ID, "v2", t2)
	mustOK(t, err)
	list, err := s.Consents(bg, a.ID)
	if err != nil || len(list) != 2 || list[0].TextVersion != "v1" || list[1].TextVersion != "v2" {
		t.Fatalf("consents: %+v %v", list, err)
	}
	if list, err = s.Consents(bg, b.ID); err != nil || list == nil || len(list) != 0 {
		t.Fatalf("other user consents: %+v %v", list, err)
	}
}

func TestImportStore(t *testing.T) {
	s, a, b, c, _, _, _ := setup(t)

	j, err := s.CreateImport(bg, a.ID, c.ID, "notes.csv")
	mustOK(t, err)
	if j.ID == "" || j.SubjectID != c.ID || j.Status != domain.ImportRunning || j.FinishedAt != nil || j.RowErrors == nil {
		t.Fatalf("create: %+v", j)
	}
	_, err = s.CreateImport(bg, b.ID, c.ID, "x.csv")
	isNF(t, "create in another user's course", err)

	_, err = s.GetImport(bg, b.ID, j.ID)
	isNF(t, "get other's import", err)
	_, err = s.GetImport(bg, a.ID, "nope")
	isNF(t, "bad import id", err)
	_, err = s.FinishImport(bg, b.ID, domain.ImportJob{ID: j.ID, Status: domain.ImportDone}, time.Now())
	isNF(t, "finish other's import", err)
	if got, _ := s.GetImport(bg, a.ID, j.ID); got.Status != domain.ImportRunning {
		t.Fatalf("other user's finish must not change the job: %+v", got)
	}
	if got, err := s.ListImports(bg, b.ID, c.ID); err != nil || got == nil || len(got) != 0 {
		t.Fatalf("other's list: %+v %v", got, err)
	}

	at := time.Date(2026, 11, 16, 10, 0, 0, 0, time.UTC)
	fin, err := s.FinishImport(bg, a.ID, domain.ImportJob{ID: j.ID, Status: domain.ImportDone, RowsTotal: 3, RowsImported: 2,
		RowErrors: []domain.ImportRowError{{Row: 3, Error: "quote not found"}}}, at)
	mustOK(t, err)
	if fin.Status != domain.ImportDone || fin.RowsTotal != 3 || fin.RowsImported != 2 || fin.FinishedAt == nil || !fin.FinishedAt.Equal(at) ||
		len(fin.RowErrors) != 1 || fin.RowErrors[0] != (domain.ImportRowError{Row: 3, Error: "quote not found"}) {
		t.Fatalf("finish: %+v", fin)
	}
	_, err = s.FinishImport(bg, a.ID, domain.ImportJob{ID: j.ID, Status: domain.ImportRunning}, at)
	if err == nil {
		t.Fatal("finishing with status running must fail")
	}
	_, err = s.FinishImport(bg, a.ID, domain.ImportJob{ID: j.ID, Status: domain.ImportDone, RowsTotal: 1, RowsImported: 2}, at)
	if err == nil {
		t.Fatal("imported > total must fail")
	}

	j2, _ := s.CreateImport(bg, a.ID, c.ID, "second.csv")
	failed, err := s.FinishImport(bg, a.ID, domain.ImportJob{ID: j2.ID, Status: domain.ImportFailed}, at)
	if err != nil || failed.Status != domain.ImportFailed || failed.RowErrors == nil {
		t.Fatalf("failed: %+v %v", failed, err)
	}
	list, err := s.ListImports(bg, a.ID, c.ID)
	if err != nil || len(list) != 2 || list[0].ID != j2.ID || list[1].ID != j.ID {
		t.Fatalf("list newest first: %+v %v", list, err)
	}
}

func TestDisputeCases(t *testing.T) {
	connect := openDB(t)
	conn := connect()
	s := store.New(conn)
	a := newUser(t, s, "A")
	c, _ := s.CreateCourse(bg, a.ID, domain.Subject{Name: "C", WeeklyHours: 2, Level: domain.LevelAdvanced, Weight: 1})
	k, _ := s.CreateConcept(bg, a.ID, c.ID, "k")
	up, _ := s.CreateUpload(bg, a.ID, domain.Upload{SubjectID: c.ID, Filename: "n"}, nil, "text")
	mustOK(t, s.SavePages(bg, a.ID, up.ID, []domain.Page{{Number: 1, RawText: "r", EditedText: "e"}}, nil))
	pages, _ := s.ListPages(bg, a.ID, up.ID)
	_, err := s.SetPageReady(bg, a.ID, pages[0].ID, []domain.SourceBlock{readyBlock("The derivative of x squared is 2x.")})
	mustOK(t, err)
	blocks, _ := s.ReadyBlocks(bg, a.ID, c.ID)
	cl, err := s.CreateClaim(bg, a.ID, k.ID, domain.ClaimInput{Text: "deriv", Kind: domain.KindStatement, Required: true,
		Anchor: domain.Anchor{BlockID: blocks[0].ID, Quote: "derivative of x squared"}})
	mustOK(t, err)

	sess, _ := s.StartSession(bg, k.ID)
	mustOK(t, s.SaveResults(bg, sess.ID, []domain.ClaimResult{
		{ClaimID: &cl.ID, StudentText: "it is 3x", Label: domain.LabelContradicted},
		{StudentText: "something else", Label: domain.LabelAbsent},
	}))
	d1, err := s.CreateDispute(bg, a.ID, domain.Dispute{SessionID: sess.ID, ClaimID: &cl.ID, Note: "I meant 2x"})
	mustOK(t, err)
	d2, _ := s.CreateDispute(bg, a.ID, domain.Dispute{SessionID: sess.ID, Note: "unmatched"})
	d3, _ := s.CreateDispute(bg, a.ID, domain.Dispute{SessionID: sess.ID, ClaimID: &cl.ID, Note: "rejected one"})

	if got, err := s.UpheldDisputes(bg); err != nil || got == nil || len(got) != 0 {
		t.Fatalf("before resolve: %+v %v", got, err)
	}
	sup, abs := domain.LabelSupported, domain.LabelAbsent
	isErr := func(what string, err error) {
		t.Helper()
		if err == nil {
			t.Errorf("%s: want error", what)
		}
	}
	isErr("upheld without label", s.ResolveDispute(bg, d1.ID, domain.DisputeUpheld, nil))
	isErr("rejected with label", s.ResolveDispute(bg, d1.ID, domain.DisputeRejected, &sup))
	isErr("unknown outcome", s.ResolveDispute(bg, d1.ID, "maybe", nil))
	isNF(t, "unknown dispute", s.ResolveDispute(bg, "00000000-0000-0000-0000-00000000dead", domain.DisputeRejected, nil))
	isNF(t, "bad dispute id", s.ResolveDispute(bg, "nope", domain.DisputeRejected, nil))

	mustOK(t, s.ResolveDispute(bg, d3.ID, domain.DisputeRejected, nil))
	mustOK(t, s.ResolveDispute(bg, d2.ID, domain.DisputeUpheld, &abs))
	mustOK(t, s.ResolveDispute(bg, d1.ID, domain.DisputeUpheld, &sup))

	cases, err := s.UpheldDisputes(bg)
	mustOK(t, err)
	if len(cases) != 2 || cases[0].DisputeID != d1.ID || cases[1].DisputeID != d2.ID {
		t.Fatalf("cases: %+v", cases)
	}
	c1 := cases[0]
	if c1.SessionID != sess.ID || c1.ConceptID != k.ID || c1.ClaimID == nil || *c1.ClaimID != cl.ID || c1.StudentText != "it is 3x" ||
		c1.CheckerLabel != domain.LabelContradicted || c1.ExpectedLabel != domain.LabelSupported || c1.Note != "I meant 2x" {
		t.Fatalf("case 1: %+v", c1)
	}
	if c2 := cases[1]; c2.ClaimID != nil || c2.StudentText != "something else" || c2.CheckerLabel != domain.LabelAbsent {
		t.Fatalf("case 2: %+v", c2)
	}
	if n := count(t, conn, `SELECT count(*) FROM disputes WHERE status = 'reviewed'`); n != 3 {
		t.Fatalf("reviewed: %d", n)
	}
}

func TestMetrics(t *testing.T) {
	connect := openDB(t)
	conn := connect()
	s := store.New(conn)
	a, b := newUser(t, s, "A"), newUser(t, s, "B")
	newUser(t, s, "C") // joined, never active
	ca, _ := s.CreateCourse(bg, a.ID, domain.Subject{Name: "A", WeeklyHours: 1, Level: domain.LevelBeginner, Weight: 1})
	cb, _ := s.CreateCourse(bg, b.ID, domain.Subject{Name: "B", WeeklyHours: 1, Level: domain.LevelBeginner, Weight: 1})
	ka, _ := s.CreateConcept(bg, a.ID, ca.ID, "ka")
	kb, _ := s.CreateConcept(bg, b.ID, cb.ID, "kb")

	mon := func(day int) time.Time { return time.Date(2026, 11, day, 0, 0, 0, 0, time.UTC) } // Nov 2 and 16 are Mondays
	finished := func(concept string, at time.Time, checked bool) string {
		t.Helper()
		var id string
		if err := conn.QueryRow(bg, `INSERT INTO teach_sessions (concept_id, status, started_at, finished_at) VALUES ($1, 'finished', $2, $2) RETURNING id::text`,
			concept, at).Scan(&id); err != nil {
			t.Fatal(err)
		}
		if checked {
			if _, err := conn.Exec(bg, `INSERT INTO review_events (event_type, concept_id, session_id, due_before, due_after, stability_before, stability_after)
				VALUES ('teach_back', $1, $2, now(), now(), 1, 1)`, concept, id); err != nil {
				t.Fatal(err)
			}
		}
		return id
	}
	// Week of Nov 2: A twice (one checked, one not), B once checked. Sunday night counts in that week.
	s1 := finished(ka.ID, mon(2).Add(10*time.Hour), true)
	finished(ka.ID, mon(8).Add(23*time.Hour+59*time.Minute), false)
	finished(kb.ID, mon(4), true)
	// Week of Nov 9: nothing. Week of Nov 16: A once, unchecked, plus an active session that must not count.
	finished(ka.ID, mon(16), false)
	if _, err := conn.Exec(bg, `INSERT INTO teach_sessions (concept_id) VALUES ($1)`, ka.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec(bg, `INSERT INTO disputes (session_id, user_id, created_at) VALUES ($1, $2, $3), ($1, $2, $4)`,
		s1, a.ID, mon(3), mon(17)); err != nil {
		t.Fatal(err)
	}

	got, err := s.WeekCounts(bg, mon(2), mon(23))
	mustOK(t, err)
	want := []domain.WeekCounts{
		{WeekStart: mon(2), ActiveUsers: 2, Sessions: 3, CheckedSessions: 2, UncheckedSessions: 1, Disputes: 1},
		{WeekStart: mon(9)},
		{WeekStart: mon(16), ActiveUsers: 1, Sessions: 1, UncheckedSessions: 1, Disputes: 1},
	}
	if len(got) != len(want) {
		t.Fatalf("weeks: %+v", got)
	}
	for i := range want {
		if !got[i].WeekStart.Equal(want[i].WeekStart) {
			t.Errorf("week %d start: %v want %v", i, got[i].WeekStart, want[i].WeekStart)
		}
		got[i].WeekStart = want[i].WeekStart
		if got[i] != want[i] {
			t.Errorf("week %d: %+v want %+v", i, got[i], want[i])
		}
	}
	if got, err := s.WeekCounts(bg, mon(9), mon(9)); err != nil || len(got) != 0 {
		t.Fatalf("empty range: %+v %v", got, err)
	}

	uw, err := s.UserWeeks(bg)
	mustOK(t, err)
	if len(uw) != 3 {
		t.Fatalf("user weeks: %+v", uw)
	}
	byID := map[string]domain.UserWeeks{}
	for _, u := range uw {
		byID[u.UserID] = u
		if u.JoinedAt.IsZero() || u.ActiveWeeks == nil {
			t.Errorf("user %s: %+v", u.UserID, u)
		}
	}
	wa := byID[a.ID].ActiveWeeks
	if len(wa) != 2 || !wa[0].Equal(mon(2)) && !wa[1].Equal(mon(2)) {
		t.Fatalf("A weeks: %v", wa)
	}
	if w := byID[b.ID].ActiveWeeks; len(w) != 1 || !w[0].Equal(mon(2)) {
		t.Fatalf("B weeks: %v", w)
	}
	for id, u := range byID {
		if id != a.ID && id != b.ID && len(u.ActiveWeeks) != 0 {
			t.Fatalf("C weeks: %v", u.ActiveWeeks)
		}
	}
}

func TestSpendLedger(t *testing.T) {
	s := store.New(openDB(t)())
	nov := time.Date(2026, 11, 30, 23, 59, 0, 0, time.UTC)
	dec := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)

	if v, err := s.MonthTotal(bg, nov); err != nil || v != 0 {
		t.Fatalf("empty month: %v %v", v, err)
	}
	if v, err := s.Add(bg, nov, 1.25); err != nil || v != 1.25 {
		t.Fatalf("first add: %v %v", v, err)
	}
	if v, err := s.Add(bg, nov.Add(-20*24*time.Hour), 0.5); err != nil || v != 1.75 {
		t.Fatalf("same month: %v %v", v, err)
	}
	if v, err := s.Add(bg, nov, 0); err != nil || v != 1.75 {
		t.Fatalf("zero add: %v %v", v, err)
	}
	if v, _ := s.Add(bg, dec, 3); v != 3 {
		t.Fatalf("new month starts at its own total: %v", v)
	}
	// A time in another zone is converted to its UTC month.
	est := time.FixedZone("est", -5*3600)
	if v, _ := s.MonthTotal(bg, time.Date(2026, 11, 30, 20, 0, 0, 0, est)); v != 3 {
		t.Fatalf("20:00 est on Nov 30 is Dec 1 UTC: %v", v)
	}
	if v, _ := s.MonthTotal(bg, nov); v != 1.75 {
		t.Fatalf("nov total: %v", v)
	}
	if _, err := s.Add(bg, nov, -0.01); err == nil {
		t.Fatal("negative spend must fail")
	}
	if v, _ := s.MonthTotal(bg, nov); v != 1.75 {
		t.Fatalf("failed add changed total: %v", v)
	}
}

func TestSpendLedgerConcurrentAndRestart(t *testing.T) {
	connect := openDB(t)
	at := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	const workers, each = 20, 0.25
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() { // each worker has its own connection, as separate server processes would
			defer wg.Done()
			_, err := store.New(connect()).Add(bg, at, each)
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		mustOK(t, err)
	}
	// A fresh connection ("restart") sees the full total.
	if v, err := store.New(connect()).MonthTotal(bg, at); err != nil || v != workers*each {
		t.Fatalf("total after concurrent adds: %v %v", v, err)
	}
}
