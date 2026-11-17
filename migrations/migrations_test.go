package migrations_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

// Needs a migrated database: `make up migrate`. Skips when DATABASE_URL is unset.
func TestReviewEventsIsAppendOnly(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set")
	}
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)

	// Everything runs in one transaction that is always rolled back, so the test leaves no rows.
	// Each rejected statement sits in a savepoint, because Postgres aborts the tx on error.
	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)

	var subjectID, conceptID, eventID string
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	// subjects.user_id defaults to the demo user, so make sure that row exists (rolled back with the rest).
	_, err = tx.Exec(ctx, `INSERT INTO users (id, name) VALUES ('00000000-0000-0000-0000-000000000001', 'demo')
		ON CONFLICT (id) DO NOTHING`)
	must(err)
	must(tx.QueryRow(ctx, `INSERT INTO subjects (name, weekly_hours, level) VALUES ('s', 1, 'beginner')
		RETURNING id`).Scan(&subjectID))
	must(tx.QueryRow(ctx, `INSERT INTO concepts (subject_id, name) VALUES ($1, 'c') RETURNING id`, subjectID).Scan(&conceptID))
	must(tx.QueryRow(ctx, `INSERT INTO review_events
		(event_type, concept_id, grade, due_before, due_after, stability_before, stability_after)
		VALUES ('teach_back', $1, 'again', now(), now(), 0, 0) RETURNING id`, conceptID).Scan(&eventID))

	rejected := map[string]struct {
		sql  string
		args []any
	}{
		"update": {`UPDATE review_events SET grade = 'good' WHERE id = $1`, []any{eventID}},
		// 0003 columns are covered by the same trigger.
		"update v2": {`UPDATE review_events SET checker_model = 'x' WHERE id = $1`, []any{eventID}},
		"delete":    {`DELETE FROM review_events WHERE id = $1`, []any{eventID}},
		"truncate":  {`TRUNCATE review_events`, nil},
	}
	for name, c := range rejected {
		sp, err := tx.Begin(ctx) // nested tx is a savepoint
		must(err)
		if _, err := sp.Exec(ctx, c.sql, c.args...); err == nil {
			t.Errorf("%s on review_events succeeded, trigger is missing", name)
		}
		sp.Rollback(ctx)
	}

	var grade string
	must(tx.QueryRow(ctx, `SELECT grade FROM review_events WHERE id = $1`, eventID).Scan(&grade))
	if grade != "again" {
		t.Fatalf("row changed: grade = %q", grade)
	}
}

// Needs a migrated database. One session may write only one teach_back event.
func TestTeachBackIsUniquePerSession(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set")
	}
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx) // leaves no rows

	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	var conceptID, sessionID, tz string
	_, err = tx.Exec(ctx, `INSERT INTO users (id, name) VALUES ('00000000-0000-0000-0000-000000000001', 'demo')
		ON CONFLICT (id) DO NOTHING`)
	must(err)
	must(tx.QueryRow(ctx, `SELECT time_zone FROM users WHERE id = '00000000-0000-0000-0000-000000000001'`).Scan(&tz))
	if tz == "" {
		t.Fatal("users.time_zone is empty")
	}
	must(tx.QueryRow(ctx, `WITH s AS (INSERT INTO subjects (name, weekly_hours, level) VALUES ('s', 1, 'beginner') RETURNING id)
		INSERT INTO concepts (subject_id, name) SELECT id, 'c' FROM s RETURNING id`).Scan(&conceptID))
	must(tx.QueryRow(ctx, `INSERT INTO teach_sessions (concept_id) VALUES ($1) RETURNING id`, conceptID).Scan(&sessionID))

	insert := func(q pgx.Tx, eventType string) error {
		_, err := q.Exec(ctx, `INSERT INTO review_events
			(event_type, concept_id, session_id, due_before, due_after, stability_before, stability_after)
			VALUES ($1, $2, $3, now(), now(), 0, 0)`, eventType, conceptID, sessionID)
		return err
	}
	must(insert(tx, "teach_back"))
	must(insert(tx, "manual_override")) // the index only covers teach_back

	sp, err := tx.Begin(ctx)
	must(err)
	if err := insert(sp, "teach_back"); err == nil {
		t.Error("second teach_back for one session succeeded, unique index is missing")
	}
	sp.Rollback(ctx)

	// Events without a session (cards, old rows) are not limited.
	for range 2 {
		_, err := tx.Exec(ctx, `INSERT INTO review_events
			(event_type, concept_id, due_before, due_after, stability_before, stability_after)
			VALUES ('teach_back', $1, now(), now(), 0, 0)`, conceptID)
		must(err)
	}
}

// Needs a migrated database. The v3 checks reject half-filled rows.
func TestContractV3Checks(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set")
	}
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx) // leaves no rows

	var subjectID, conceptID, sourceID string
	if err := tx.QueryRow(ctx, `WITH u AS (INSERT INTO users (name) VALUES ('v3') RETURNING id)
		INSERT INTO subjects (user_id, name, weekly_hours, level) SELECT id, 's', 1, 'beginner' FROM u RETURNING id`).Scan(&subjectID); err != nil {
		t.Fatal(err)
	}
	if err := tx.QueryRow(ctx, `INSERT INTO concepts (subject_id, name) VALUES ($1, 'c') RETURNING id`, subjectID).Scan(&conceptID); err != nil {
		t.Fatal(err)
	}
	if err := tx.QueryRow(ctx, `INSERT INTO sources (kind, title, text, subject_id) VALUES ('course', 't', '', $1) RETURNING id`, subjectID).Scan(&sourceID); err != nil {
		t.Fatal(err)
	}

	ok := map[string]string{
		"phase 0 claim": `INSERT INTO claims (concept_id, text, source_id) VALUES ('` + conceptID + `', 't', '` + sourceID + `')`,
		"pasted upload": `INSERT INTO uploads (subject_id, filename, pasted_text) VALUES ('` + subjectID + `', 'f', 'x')`,
		"unused invite": `INSERT INTO invites (token_hash, name) VALUES ('\x01', 'n')`,
	}
	bad := map[string]string{
		"anchor without quote": `INSERT INTO claims (concept_id, text, source_id, anchor_block_id) VALUES ('` + conceptID + `', 't', '` + sourceID + `', gen_random_uuid())`,
		"unknown kind":         `INSERT INTO claims (concept_id, text, source_id, kind) VALUES ('` + conceptID + `', 't', '` + sourceID + `', 'vibe')`,
		"upload with neither":  `INSERT INTO uploads (subject_id, filename) VALUES ('` + subjectID + `', 'f')`,
		"upload with both":     `INSERT INTO uploads (subject_id, filename, pdf, pasted_text) VALUES ('` + subjectID + `', 'f', '\x00', 'x')`,
		"used invite no user":  `INSERT INTO invites (token_hash, name, used_at) VALUES ('\x02', 'n', now())`,
		"bad upload status":    `INSERT INTO uploads (subject_id, filename, pasted_text, status) VALUES ('` + subjectID + `', 'f', 'x', 'done')`,
	}
	for name, q := range ok {
		if _, err := tx.Exec(ctx, q); err != nil {
			t.Errorf("%s rejected: %v", name, err)
		}
	}
	for name, q := range bad {
		sp, err := tx.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := sp.Exec(ctx, q); err == nil {
			t.Errorf("%s accepted", name)
		}
		sp.Rollback(ctx)
	}
}

// Needs a migrated database. D43: one transaction that sets app.delete_user may
// delete that user's review_events, and nothing else.
func TestReviewEventsDeleteAccountGuard(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set")
	}
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx) // leaves no rows

	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	// One user with one event each. Returns the user id and the event id.
	seed := func(name string) (user, event string) {
		must(tx.QueryRow(ctx, `INSERT INTO users (name) VALUES ($1) RETURNING id`, name).Scan(&user))
		must(tx.QueryRow(ctx, `WITH s AS (INSERT INTO subjects (user_id, name, weekly_hours, level) VALUES ($1, 's', 1, 'beginner') RETURNING id),
			c AS (INSERT INTO concepts (subject_id, name) SELECT id, 'c' FROM s RETURNING id)
			INSERT INTO review_events (event_type, concept_id, grade, due_before, due_after, stability_before, stability_after)
			SELECT 'teach_back', id, 'again', now(), now(), 0, 0 FROM c RETURNING id`, user).Scan(&event))
		return user, event
	}
	alice, aliceEvent := seed("alice")
	_, bobEvent := seed("bob")

	rejected := func(name, sql string, args ...any) {
		t.Helper()
		sp, err := tx.Begin(ctx)
		must(err)
		if _, err := sp.Exec(ctx, sql, args...); err == nil {
			t.Errorf("%s succeeded", name)
		}
		sp.Rollback(ctx)
	}
	rejected("delete without the setting", `DELETE FROM review_events WHERE id = $1`, aliceEvent)

	_, err = tx.Exec(ctx, `SELECT set_config('app.delete_user', $1, true)`, alice) // same as SET LOCAL
	must(err)
	rejected("delete of another user's event", `DELETE FROM review_events WHERE id = $1`, bobEvent)
	rejected("update of the user's own event", `UPDATE review_events SET grade = 'good' WHERE id = $1`, aliceEvent)
	rejected("truncate", `TRUNCATE review_events`)

	tag, err := tx.Exec(ctx, `DELETE FROM review_events WHERE id = $1`, aliceEvent)
	must(err)
	if tag.RowsAffected() != 1 {
		t.Fatalf("deleted %d rows, want 1", tag.RowsAffected())
	}
	var n int
	must(tx.QueryRow(ctx, `SELECT count(*) FROM review_events WHERE id = $1`, bobEvent).Scan(&n))
	if n != 1 {
		t.Fatal("bob's event is gone")
	}
}

// Needs a migrated database. The v4 checks reject half-filled rows.
func TestContractV4Checks(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set")
	}
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx) // leaves no rows

	var userID, subjectID, sessionID string
	if err := tx.QueryRow(ctx, `INSERT INTO users (name) VALUES ('v4') RETURNING id`).Scan(&userID); err != nil {
		t.Fatal(err)
	}
	if err := tx.QueryRow(ctx, `INSERT INTO subjects (user_id, name, weekly_hours, level) VALUES ($1, 's', 1, 'beginner') RETURNING id`, userID).Scan(&subjectID); err != nil {
		t.Fatal(err)
	}
	if err := tx.QueryRow(ctx, `WITH c AS (INSERT INTO concepts (subject_id, name) VALUES ($1, 'c') RETURNING id)
		INSERT INTO teach_sessions (concept_id) SELECT id FROM c RETURNING id`, subjectID).Scan(&sessionID); err != nil {
		t.Fatal(err)
	}
	u, s, ss := "'"+userID+"'", "'"+subjectID+"'", "'"+sessionID+"'"
	ok := map[string]string{
		"feedback":        `INSERT INTO feedback_answers (user_id, week_start, question, answer) VALUES (` + u + `, '2026-11-16', 'q', 'a')`,
		"consent":         `INSERT INTO consents (user_id, text_version) VALUES (` + u + `, 'v1')`,
		"running import":  `INSERT INTO import_jobs (subject_id, filename) VALUES (` + s + `, 'f.csv')`,
		"finished import": `INSERT INTO import_jobs (subject_id, filename, status, rows_total, rows_imported, row_errors, finished_at) VALUES (` + s + `, 'f.csv', 'done', 2, 1, '[{"row":3,"error":"x"}]', now())`,
		"spend":           `INSERT INTO model_spend (month, usd) VALUES ('2026-11-01', 1.25)`,
		"upheld":          `INSERT INTO disputes (session_id, user_id, status, outcome, expected_label) VALUES (` + ss + `, ` + u + `, 'reviewed', 'upheld', 'partial')`,
		"rejected":        `INSERT INTO disputes (session_id, user_id, status, outcome) VALUES (` + ss + `, ` + u + `, 'reviewed', 'rejected')`,
		"open dispute":    `INSERT INTO disputes (session_id, user_id) VALUES (` + ss + `, ` + u + `)`,
	}
	bad := map[string]string{
		"second feedback same week": `INSERT INTO feedback_answers (user_id, week_start, question, answer) VALUES (` + u + `, '2026-11-16', 'q', 'b')`,
		"empty consent version":     `INSERT INTO consents (user_id, text_version) VALUES (` + u + `, '')`,
		"done import not finished":  `INSERT INTO import_jobs (subject_id, filename, status) VALUES (` + s + `, 'f.csv', 'done')`,
		"more imported than total":  `INSERT INTO import_jobs (subject_id, filename, rows_total, rows_imported) VALUES (` + s + `, 'f.csv', 1, 2)`,
		"row errors not a list":     `INSERT INTO import_jobs (subject_id, filename, row_errors) VALUES (` + s + `, 'f.csv', '{}')`,
		"spend mid month":           `INSERT INTO model_spend (month, usd) VALUES ('2026-11-02', 1)`,
		"negative spend":            `INSERT INTO model_spend (month, usd) VALUES ('2026-10-01', -1)`,
		"upheld without label":      `INSERT INTO disputes (session_id, user_id, status, outcome) VALUES (` + ss + `, ` + u + `, 'reviewed', 'upheld')`,
		"rejected with label":       `INSERT INTO disputes (session_id, user_id, status, outcome, expected_label) VALUES (` + ss + `, ` + u + `, 'reviewed', 'rejected', 'partial')`,
		"outcome while open":        `INSERT INTO disputes (session_id, user_id, outcome) VALUES (` + ss + `, ` + u + `, 'rejected')`,
	}
	// ok runs first so the duplicate feedback row has something to collide with.
	for name, q := range ok {
		if _, err := tx.Exec(ctx, q); err != nil {
			t.Errorf("%s rejected: %v", name, err)
		}
	}
	for name, q := range bad {
		sp, err := tx.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := sp.Exec(ctx, q); err == nil {
			t.Errorf("%s accepted", name)
		}
		sp.Rollback(ctx)
	}
}

// Needs a migrated database. The v5 checks keep billing and share rows whole,
// and deleting a user takes their entitlement and share links with them.
func TestContractV5Checks(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set")
	}
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx) // leaves no rows

	var userID, subjectID string
	var instructor bool
	if err := tx.QueryRow(ctx, `INSERT INTO users (name) VALUES ('v5') RETURNING id, is_instructor`).Scan(&userID, &instructor); err != nil {
		t.Fatal(err)
	}
	if instructor {
		t.Fatal("new user is an instructor by default")
	}
	if err := tx.QueryRow(ctx, `INSERT INTO subjects (user_id, name, weekly_hours, level) VALUES ($1, 's', 1, 'beginner') RETURNING id`, userID).Scan(&subjectID); err != nil {
		t.Fatal(err)
	}
	u, s := "'"+userID+"'", "'"+subjectID+"'"
	hash := `'\x` + strings.Repeat("ab", 32) + `'`
	ok := map[string]string{
		"month": `INSERT INTO entitlements (user_id, plan, status, period_end, customer_id, subscription_id) VALUES (` + u + `, 'month', 'active', now(), 'cus_1', 'sub_1')`,
		"event": `INSERT INTO billing_events (event_id, event_type, payload) VALUES ('evt_1', 'checkout.session.completed', '{}')`,
		"share": `INSERT INTO shared_packs (owner_id, subject_id, token_hash) VALUES (` + u + `, ` + s + `, ` + hash + `)`,
	}
	bad := map[string]string{
		"second entitlement":     `INSERT INTO entitlements (user_id, plan, status, period_end, customer_id) VALUES (` + u + `, 'term', 'active', now(), 'cus_2')`,
		"month without sub":      `INSERT INTO entitlements (user_id, plan, status, period_end, customer_id) SELECT id, 'month', 'active', now(), 'cus_3' FROM users WHERE id <> ` + u + ` LIMIT 1`,
		"unknown plan":           `INSERT INTO entitlements (user_id, plan, status, period_end, customer_id) SELECT id, 'year', 'active', now(), 'cus_4' FROM users WHERE id <> ` + u + ` LIMIT 1`,
		"replayed event":         `INSERT INTO billing_events (event_id, event_type, payload) VALUES ('evt_1', 'x', '{}')`,
		"payload not an object":  `INSERT INTO billing_events (event_id, event_type, payload) VALUES ('evt_2', 'x', '[]')`,
		"same token twice":       `INSERT INTO shared_packs (owner_id, subject_id, token_hash) VALUES (` + u + `, ` + s + `, ` + hash + `)`,
		"short token hash":       `INSERT INTO shared_packs (owner_id, subject_id, token_hash) VALUES (` + u + `, ` + s + `, '\xab')`,
		"revoked before created": `INSERT INTO shared_packs (owner_id, subject_id, token_hash, created_at, revoked_at) VALUES (` + u + `, ` + s + `, '\x` + strings.Repeat("cd", 32) + `', now(), now() - interval '1 day')`,
	}
	// ok runs first so the duplicates have something to collide with.
	for name, q := range ok {
		if _, err := tx.Exec(ctx, q); err != nil {
			t.Errorf("%s rejected: %v", name, err)
		}
	}
	for name, q := range bad {
		sp, err := tx.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if tag, err := sp.Exec(ctx, q); err == nil && tag.RowsAffected() > 0 {
			t.Errorf("%s accepted", name)
		}
		sp.Rollback(ctx)
	}

	// The delete-account path removes subjects then the user; v5 rows must not block it.
	if _, err := tx.Exec(ctx, `DELETE FROM subjects WHERE user_id = $1`, userID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID); err != nil {
		t.Fatalf("v5 rows block account delete: %v", err)
	}
	var left int
	if err := tx.QueryRow(ctx, `SELECT (SELECT count(*) FROM entitlements WHERE user_id = $1) + (SELECT count(*) FROM shared_packs WHERE owner_id = $1)`, userID).Scan(&left); err != nil || left != 0 {
		t.Fatalf("rows left after delete: %d %v", left, err)
	}
}
