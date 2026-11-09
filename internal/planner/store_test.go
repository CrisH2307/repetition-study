package planner

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/CrisH2307/repetition-study/internal/domain"
	"github.com/CrisH2307/repetition-study/internal/fakes"
)

// testTx returns a Store on a transaction that is rolled back at the end.
// review_events is append-only, so rolling back is the only way to stay clean.
func testTx(t *testing.T) (*Store, pgx.Tx, string) {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://repetition:repetition@localhost:5432/repetition?sslmode=disable"
	}
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Skipf("no database (run make up migrate): %v", err)
	}
	t.Cleanup(func() { _ = conn.Close(ctx) })
	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tx.Rollback(ctx) })
	var uid string
	if err := tx.QueryRow(ctx, `INSERT INTO users (name) VALUES ('planner test') RETURNING id::text`).Scan(&uid); err != nil {
		t.Skipf("schema missing (run make migrate): %v", err)
	}
	st := New(tx, fakes.NewClock())
	st.UserID = uid
	return st, tx, uid
}

func one(t *testing.T, tx pgx.Tx, sql string, args ...any) string {
	t.Helper()
	var id string
	if err := tx.QueryRow(context.Background(), sql, args...).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func mustExec(t *testing.T, tx pgx.Tx, sql string, args ...any) {
	t.Helper()
	if _, err := tx.Exec(context.Background(), sql, args...); err != nil {
		t.Fatal(err)
	}
}

// seedSubject adds a subject with n concepts, each scheduled with the given stability.
func seedSubject(t *testing.T, tx pgx.Tx, uid, name string, hours float64, n int, stability float64) (string, []string) {
	t.Helper()
	sid := one(t, tx, `INSERT INTO subjects (user_id, name, weekly_hours, level, exam_date) VALUES ($1, $2, $3, 'beginner', '2026-12-15') RETURNING id::text`, uid, name, hours)
	var cids []string
	for i := 0; i < n; i++ {
		cid := one(t, tx, `INSERT INTO concepts (subject_id, name) VALUES ($1, $2) RETURNING id::text`, sid, name+" concept "+string(rune('a'+i)))
		mustExec(t, tx, `INSERT INTO concept_schedules (concept_id, due_at, stability, difficulty, state, reps, last_review_at)
			VALUES ($1, $2, $3, 5, 2, 3, $4)`, cid, fakes.Epoch.AddDate(0, 0, 3), stability, fakes.Epoch.AddDate(0, 0, -4))
		cids = append(cids, cid)
	}
	return sid, cids
}

func blocksFor(p domain.Plan, subjectID string) int {
	return count(p.Blocks, subjectID)
}

func TestGenerateKeepsLockedAndIsRepeatable(t *testing.T) {
	st, tx, uid := testTx(t)
	ctx := context.Background()
	a, cs := seedSubject(t, tx, uid, "A", 4, 3, 5)
	seedSubject(t, tx, uid, "B", 3, 3, 5)
	lockedAt := week.Add(10 * time.Hour)
	lid := one(t, tx, `INSERT INTO plan_blocks (subject_id, concept_id, kind, starts_at, minutes, reason, locked)
		VALUES ($1, $2, 'review', $3, 30, 'pinned by student', true) RETURNING id::text`, a, cs[0], lockedAt)

	p1, err := st.Generate(ctx, uid, week)
	if err != nil {
		t.Fatal(err)
	}
	p2, err := st.Generate(ctx, uid, week) // regenerating must replace, not append
	if err != nil {
		t.Fatal(err)
	}
	if len(p1.Blocks) != len(p2.Blocks) {
		t.Fatalf("regenerate changed block count: %d then %d", len(p1.Blocks), len(p2.Blocks))
	}
	var kept bool
	for _, b := range p2.Blocks {
		if b.ID == lid {
			kept = b.Locked && b.StartsAt.Equal(lockedAt) && b.Reason == "pinned by student"
		} else if b.StartsAt.Equal(lockedAt) {
			t.Fatal("a generated block took the locked slot")
		}
		if b.Reason == "" {
			t.Fatalf("block without reason: %+v", b)
		}
	}
	if !kept {
		t.Fatal("locked block was moved or dropped")
	}
	if n := blocksFor(p2, a); n > 8 {
		t.Fatalf("subject A has %d blocks, ceiling is 8", n)
	}
	var stored int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM plan_blocks p JOIN subjects su ON su.id = p.subject_id WHERE su.user_id = $1`, uid).Scan(&stored); err != nil || stored != len(p2.Blocks) {
		t.Fatalf("stored %d blocks, plan has %d (err %v)", stored, len(p2.Blocks), err)
	}
}

// The demo loop: a wrong teach-back lowers stability, which raises next week's hours,
// and Further Planning shows recall and explanation apart.
func TestWrongExplanationRaisesNextWeekHours(t *testing.T) {
	st, tx, uid := testTx(t)
	ctx := context.Background()
	a, cs := seedSubject(t, tx, uid, "A", 6, 3, 30)
	b, _ := seedSubject(t, tx, uid, "B", 6, 3, 30)

	hoursOf := func(r Report, id string) float64 {
		for _, s := range r.Subjects {
			if s.SubjectID == id {
				return s.NextWeekHours
			}
		}
		t.Fatalf("subject %s missing", id)
		return 0
	}
	// This week's plan is stored before the teach-back, so the report can show the change.
	if _, err := st.Generate(ctx, uid, week.AddDate(0, 0, -7)); err != nil {
		t.Fatal(err)
	}
	before, err := st.Progress(ctx, week.AddDate(0, 0, -7))
	if err != nil {
		t.Fatal(err)
	}

	// A wrong explanation of concept 0: one required claim supported, one contradicted with a citation.
	src := one(t, tx, `INSERT INTO sources (kind, title, text) VALUES ('course', 'Notes', 'The derivative of x^2 is 2x.') RETURNING id::text`)
	c1 := one(t, tx, `INSERT INTO claims (concept_id, text, source_id, required) VALUES ($1, 'd/dx x^2 = 2x', $2, true) RETURNING id::text`, cs[0], src)
	c2 := one(t, tx, `INSERT INTO claims (concept_id, text, source_id, required) VALUES ($1, 'power rule lowers the exponent', $2, true) RETURNING id::text`, cs[0], src)
	ses := one(t, tx, `INSERT INTO teach_sessions (concept_id, status) VALUES ($1, 'finished') RETURNING id::text`, cs[0])
	mustExec(t, tx, `INSERT INTO claim_labels (session_id, claim_id, student_text, label) VALUES ($1, $2, 'it is 2x', 'supported')`, ses, c1)
	mustExec(t, tx, `INSERT INTO claim_labels (session_id, claim_id, student_text, label, citation_source_id, citation_quote, citation_locator, correction)
		VALUES ($1, $2, 'the exponent goes up', 'contradicted', $3, 'The derivative of x^2 is 2x.', 'p1', 'The exponent drops by one.')`, ses, c2, src)
	// What the scheduler would write: Again drops stability for every concept the student got wrong.
	mustExec(t, tx, `UPDATE concept_schedules SET stability = 1 WHERE concept_id = $1`, cs[0])
	mustExec(t, tx, `INSERT INTO review_events (event_type, concept_id, grade, due_before, due_after, stability_before, stability_after)
		VALUES ('teach_back', $1, 'again', $2, $3, 30, 1)`, cs[0], fakes.Epoch, fakes.Epoch.AddDate(0, 0, 1))
	mustExec(t, tx, `INSERT INTO review_events (event_type, concept_id, grade, due_before, due_after, stability_before, stability_after)
		VALUES ('card_review', $1, 'good', $2, $3, 1, 2), ('card_review', $1, 'again', $2, $3, 2, 1)`, cs[0], fakes.Epoch, fakes.Epoch.AddDate(0, 0, 1))

	after, err := st.Progress(ctx, week.AddDate(0, 0, -7))
	if err != nil {
		t.Fatal(err)
	}
	if hoursOf(after, a) <= hoursOf(before, a) {
		t.Fatalf("subject A next week hours %.1f -> %.1f, want an increase", hoursOf(before, a), hoursOf(after, a))
	}
	if hoursOf(after, b) != hoursOf(before, b) {
		t.Fatalf("subject B changed: %.1f -> %.1f", hoursOf(before, b), hoursOf(after, b))
	}

	var sa SubjectReport
	for _, s := range after.Subjects {
		if s.SubjectID == a {
			sa = s
		}
	}
	if sa.Change <= 0 || sa.Reason == "" {
		t.Fatalf("change %.1f reason %q", sa.Change, sa.Reason)
	}
	var c0, c1r *ConceptReport
	for i := range sa.Concepts {
		if sa.Concepts[i].ConceptID == cs[0] {
			c0 = &sa.Concepts[i]
		} else if c1r == nil {
			c1r = &sa.Concepts[i]
		}
	}
	if c0.Explanation == nil || *c0.Explanation != 0.5 {
		t.Fatalf("explanation = %v, want 0.5", c0.Explanation)
	}
	if c0.Recall == nil || *c0.Recall != 0.5 || c0.CardReviews != 2 {
		t.Fatalf("recall = %v over %d reviews, want 0.5 over 2", c0.Recall, c0.CardReviews)
	}
	if c0.Retrievability == nil || len(c0.Gaps) != 1 || c0.Gaps[0].Citation == nil || c0.Gaps[0].Label != domain.LabelContradicted {
		t.Fatalf("want one cited contradicted gap and a retrievability, got %+v", c0)
	}
	if c1r.Recall != nil || c1r.Explanation != nil || len(c1r.Gaps) != 0 {
		t.Fatalf("concept with no events must report no data, got %+v", c1r)
	}

	fp, err := st.FurtherPlanning(ctx, week.AddDate(0, 0, -7))
	if err != nil || len(fp.Subjects) != 2 {
		t.Fatalf("FurtherPlanning: %v, %d subjects", err, len(fp.Subjects))
	}
}
