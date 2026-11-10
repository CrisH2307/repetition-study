package migrations_test

import (
	"context"
	"os"
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
