package store_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/CrisH2307/repetition-study/internal/domain"
	"github.com/CrisH2307/repetition-study/internal/store"
)

// privacyTables are every table that holds user data, with the number of rows
// one populated user leaves behind.
var privacyTables = map[string]int{
	"subjects": 1, "concepts": 1, "claims": 1, "sources": 1, "source_blocks": 1, "uploads": 1, "pages": 1,
	"teach_sessions": 1, "teach_turns": 1, "taught_claims": 1, "claim_labels": 1, "self_grades": 1, "disputes": 1,
	"cards": 1, "concept_schedules": 1, "plan_blocks": 1, "review_events": 1, "import_jobs": 1,
	"feedback_answers": 1, "consents": 1, "digests": 1, "auth_sessions": 1, "invites": 1, "users": 1,
}

// populate gives the user one row in every table above. Every text carries the
// name, so a leak shows up in a plain string search.
func populate(t *testing.T, s *store.Store, conn *pgx.Conn, u domain.User) {
	t.Helper()
	n := u.Name
	c, err := s.CreateCourse(bg, u.ID, domain.Subject{Name: "course-" + n, WeeklyHours: 2, Level: domain.LevelBeginner, Weight: 1})
	mustOK(t, err)
	k, err := s.CreateConcept(bg, u.ID, c.ID, "concept-"+n)
	mustOK(t, err)
	up, err := s.CreateUpload(bg, u.ID, domain.Upload{SubjectID: c.ID, Filename: "file-" + n}, nil, "The slope is rise over run "+n)
	mustOK(t, err)
	text := "The slope is rise over run " + n + "."
	mustOK(t, s.SavePages(bg, u.ID, up.ID, []domain.Page{{Number: 1, RawText: text, EditedText: text}}, [][]byte{[]byte("png")}))
	pages, err := s.ListPages(bg, u.ID, up.ID)
	mustOK(t, err)
	_, err = s.SetPageReady(bg, u.ID, pages[0].ID, []domain.SourceBlock{readyBlock(text)})
	mustOK(t, err)
	blocks, err := s.ReadyBlocks(bg, u.ID, c.ID)
	mustOK(t, err)
	cl, err := s.CreateClaim(bg, u.ID, k.ID, domain.ClaimInput{Text: "claim-" + n, Kind: domain.KindStatement, Required: true,
		Anchor: domain.Anchor{BlockID: blocks[0].ID, Quote: "slope is rise over run"}})
	mustOK(t, err)
	srcID := *mustUpload(t, s, u.ID, up.ID).SourceID

	sess, err := s.StartSession(bg, k.ID)
	mustOK(t, err)
	_, err = s.SaveTurn(bg, domain.TaughtClaim{SessionID: sess.ID, ConceptID: k.ID, StudentText: "taught-" + n, TurnNo: 1}, "transcript-"+n, `\latex`)
	mustOK(t, err)
	corr, quote := "fix-"+n, "slope is rise over run"
	mustOK(t, s.SaveResults(bg, sess.ID, []domain.ClaimResult{{ClaimID: &cl.ID, StudentText: "taught-" + n, Label: domain.LabelContradicted,
		Citation: &domain.Citation{SourceID: srcID, Quote: quote, Locator: "p1"}, Correction: &corr}}))
	mustOK(t, s.SaveSelfGrades(bg, u.ID, sess.ID, []domain.SelfGrade{{ClaimID: cl.ID, Rating: domain.RatingWrong}}))
	_, err = s.CreateDispute(bg, u.ID, domain.Dispute{SessionID: sess.ID, ClaimID: &cl.ID, Note: "dispute-" + n})
	mustOK(t, err)

	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := conn.Exec(bg, sql, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec(`INSERT INTO cards (concept_id, front, back) VALUES ($1, 'f', 'b')`, k.ID)
	exec(`INSERT INTO concept_schedules (concept_id, due_at) VALUES ($1, now())`, k.ID)
	exec(`INSERT INTO plan_blocks (subject_id, concept_id, kind, starts_at, minutes) VALUES ($1, $2, 'teach', now(), 15)`, c.ID, k.ID)
	exec(`INSERT INTO review_events (event_type, concept_id, session_id, claim_id, due_before, due_after, stability_before, stability_after, payload)
		VALUES ('teach_back', $1, $2, $3, now(), now() + interval '1 day', 1, 2, $4)`, k.ID, sess.ID, cl.ID, `{"who":"`+n+`"}`)
	exec(`INSERT INTO import_jobs (subject_id, filename) VALUES ($1, $2)`, c.ID, "import-"+n)
	exec(`INSERT INTO feedback_answers (user_id, week_start, question, answer) VALUES ($1, '2026-11-16', 'q', $2)`, u.ID, "answer-"+n)
	exec(`INSERT INTO consents (user_id, text_version) VALUES ($1, 'v1')`, u.ID)
	exec(`INSERT INTO digests (user_id, digest_date) VALUES ($1, '2026-11-16')`, u.ID)
	mustOK(t, s.CreateSession(bg, u.ID, []byte("tok-"+n), time.Now().Add(time.Hour)))
}

func mustUpload(t *testing.T, s *store.Store, userID, id string) domain.Upload {
	t.Helper()
	u, err := s.GetUpload(bg, userID, id)
	mustOK(t, err)
	if u.SourceID == nil {
		t.Fatal("upload has no source")
	}
	return u
}

func tableCounts(t *testing.T, conn *pgx.Conn) map[string]int {
	t.Helper()
	out := map[string]int{}
	for tb := range privacyTables {
		out[tb] = count(t, conn, `SELECT count(*) FROM `+tb)
	}
	return out
}

func TestPrivacyExportAndDelete(t *testing.T) {
	connect := openDB(t)
	conn := connect()
	s := store.New(conn)
	// A shared allowlist source belongs to nobody and must survive.
	if _, err := conn.Exec(bg, `INSERT INTO sources (kind, title, text) VALUES ('allowlist', 'shared', 'x')`); err != nil {
		t.Fatal(err)
	}
	a, b, c := newUser(t, s, "alice"), newUser(t, s, "bob"), newUser(t, s, "carol")
	populate(t, s, conn, a)
	populate(t, s, conn, b)
	// carol has no data at all: delete must still work and export must be empty lists.
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)

	// ---- export ----
	exA, err := s.ExportUser(bg, a.ID, now)
	mustOK(t, err)
	exB, err := s.ExportUser(bg, b.ID, now)
	mustOK(t, err)
	raw, _ := json.Marshal(exA)
	for _, leak := range []string{b.ID, "bob", exB.Courses[0].ID, exB.Concepts[0].ID, exB.ReviewEvents[0].ID, exB.Sessions[0].ID} {
		if strings.Contains(string(raw), leak) {
			t.Errorf("export of alice contains %q from bob", leak)
		}
	}
	if exA.User != a || !exA.ExportedAt.Equal(now) {
		t.Errorf("user/time: %+v %v", exA.User, exA.ExportedAt)
	}
	for name, n := range map[string]int{
		"consents": len(exA.Consents), "feedback": len(exA.Feedback), "courses": len(exA.Courses), "concepts": len(exA.Concepts),
		"uploads": len(exA.Uploads), "pages": len(exA.Pages), "sessions": len(exA.Sessions), "turns": len(exA.Turns),
		"taught": len(exA.TaughtClaims), "reports": len(exA.Reports), "self_grades": len(exA.SelfGrades),
		"disputes": len(exA.Disputes), "events": len(exA.ReviewEvents), "imports": len(exA.Imports),
	} {
		if n != 1 {
			t.Errorf("export %s: %d rows, want 1", name, n)
		}
	}
	if len(exA.Concepts[0].Claims) != 1 || exA.Concepts[0].Claims[0].Text != "claim-alice" {
		t.Errorf("concept claims: %+v", exA.Concepts[0].Claims)
	}
	if r := exA.Reports[0]; r.Status != domain.ReportChecked || len(r.Results) != 1 || r.Results[0].Citation == nil || *r.Results[0].Correction != "fix-alice" {
		t.Errorf("report: %+v", r)
	}
	if !strings.Contains(string(exA.ReviewEvents[0].Payload), "alice") || exA.Turns[0].Transcript != "transcript-alice" || exA.Pages[0].HasImage != true {
		t.Errorf("event/turn/page: %+v %+v %+v", exA.ReviewEvents[0], exA.Turns[0], exA.Pages[0])
	}
	exC, err := s.ExportUser(bg, c.ID, now)
	mustOK(t, err)
	if rawC, _ := json.Marshal(exC); strings.Contains(string(rawC), "null") {
		t.Errorf("empty export has null lists: %s", rawC)
	}
	_, err = s.ExportUser(bg, "00000000-0000-0000-0000-0000000000ff", now)
	isNF(t, "export unknown user", err)
	_, err = s.ExportUser(bg, "not-a-uuid", now)
	isNF(t, "export bad id", err)

	// ---- delete ----
	before := tableCounts(t, conn)
	for tb, per := range privacyTables {
		if before[tb] != per*2+map[string]int{"users": 1, "invites": 1, "sources": 1}[tb] {
			t.Fatalf("setup: %s has %d rows", tb, before[tb])
		}
	}
	// A failing step rolls everything back, review history included: a claim of
	// bob's cites alice's source, so alice's sources cannot be removed.
	if _, err := conn.Exec(bg, `INSERT INTO claims (concept_id, text, source_id) VALUES ($1, 'x', $2)`,
		exB.Concepts[0].ID, mustUpload(t, s, a.ID, exA.Uploads[0].ID).SourceID); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteUser(bg, a.ID); err == nil {
		t.Fatal("delete should fail while another user's claim cites alice's source")
	}
	if n := count(t, conn, `SELECT count(*) FROM review_events`); n != 2 {
		t.Fatalf("failed delete lost review events: %d", n)
	}
	if _, err := s.ExportUser(bg, a.ID, now); err != nil {
		t.Fatalf("alice gone after failed delete: %v", err)
	}
	if _, err := conn.Exec(bg, `DELETE FROM claims WHERE text = 'x'`); err != nil {
		t.Fatal(err)
	}

	mustOK(t, s.DeleteUser(bg, a.ID))
	after := tableCounts(t, conn)
	for tb, per := range privacyTables {
		if want := before[tb] - per - map[string]int{"invites": 0}[tb]; after[tb] != want {
			t.Errorf("%s: %d rows after delete, want %d", tb, after[tb], want)
		}
	}
	if n := count(t, conn, `SELECT count(*) FROM sources WHERE title = 'shared'`); n != 1 {
		t.Errorf("shared source removed")
	}
	for _, tb := range []string{"subjects", "auth_sessions", "digests", "feedback_answers", "consents", "disputes", "self_grades"} {
		sql := `SELECT count(*) FROM ` + tb + ` WHERE user_id = $1`
		if n := count(t, conn, sql, a.ID); n != 0 {
			t.Errorf("%s still has rows of alice", tb)
		}
	}

	// Bob is untouched, byte for byte.
	exB2, err := s.ExportUser(bg, b.ID, now)
	mustOK(t, err)
	j1, _ := json.Marshal(exB)
	j2, _ := json.Marshal(exB2)
	if string(j1) != string(j2) {
		t.Errorf("bob changed:\n%s\n%s", j1, j2)
	}
	if _, err := s.ExportUser(bg, a.ID, now); err == nil {
		t.Error("alice still exports")
	}
	// Deleting again, or a user that never existed, is ErrNotFound and changes nothing.
	isNF(t, "re-delete", s.DeleteUser(bg, a.ID))
	isNF(t, "unknown user", s.DeleteUser(bg, "00000000-0000-0000-0000-0000000000ff"))
	isNF(t, "bad id", s.DeleteUser(bg, "nope"))

	// The setting ended with the transaction: review_events still rejects every
	// delete outside DeleteUser, even for bob's own events.
	if v := fmt.Sprint(queryString(t, conn, `SELECT coalesce(current_setting('app.delete_user', true), '')`)); v != "" {
		t.Errorf("app.delete_user leaked: %q", v)
	}
	if _, err := conn.Exec(bg, `DELETE FROM review_events`); err == nil {
		t.Error("review_events delete succeeded outside DeleteUser")
	}
	if n := count(t, conn, `SELECT count(*) FROM review_events`); n != 1 {
		t.Errorf("review_events: %d, want bob's 1", n)
	}

	// Carol, with no data, deletes cleanly, and bob still has everything.
	mustOK(t, s.DeleteUser(bg, c.ID))
	mustOK(t, s.DeleteUser(bg, b.ID))
	for tb, n := range tableCounts(t, conn) {
		if want := map[string]int{"sources": 1}[tb]; n != want {
			t.Errorf("%s: %d rows left after every user is gone, want %d", tb, n, want)
		}
	}
}

func queryString(t *testing.T, conn *pgx.Conn, sql string) string {
	t.Helper()
	var v string
	if err := conn.QueryRow(bg, sql).Scan(&v); err != nil {
		t.Fatal(err)
	}
	return v
}
