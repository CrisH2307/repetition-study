package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/CrisH2307/repetition-study/internal/domain"
)

var (
	_ domain.FeedbackStore = (*Store)(nil)
	_ domain.ImportStore   = (*Store)(nil)
	_ domain.EvalCaseStore = (*Store)(nil)
	_ domain.MetricsStore  = (*Store)(nil)
	_ domain.SpendLedger   = (*Store)(nil)
)

// Feedback and consent (D40, D44).

func (s *Store) SaveFeedback(ctx context.Context, userID string, a domain.FeedbackAnswer) (domain.FeedbackAnswer, error) {
	var out domain.FeedbackAnswer
	err := s.pool.QueryRow(ctx, `INSERT INTO feedback_answers (user_id, week_start, question, answer) VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, week_start) DO UPDATE SET question = EXCLUDED.question, answer = EXCLUDED.answer
		RETURNING week_start, question, answer, created_at`, userID, a.WeekStart, a.Question, a.Answer).
		Scan(&out.WeekStart, &out.Question, &out.Answer, &out.CreatedAt)
	return out, nf(err)
}

func (s *Store) GetFeedback(ctx context.Context, userID string, weekStart time.Time) (domain.FeedbackAnswer, error) {
	var out domain.FeedbackAnswer
	err := s.pool.QueryRow(ctx, `SELECT week_start, question, answer, created_at FROM feedback_answers
		WHERE user_id = $1 AND week_start = $2`, userID, weekStart).
		Scan(&out.WeekStart, &out.Question, &out.Answer, &out.CreatedAt)
	return out, nf(err)
}

// SaveConsent keeps the first accepted_at for a text version.
func (s *Store) SaveConsent(ctx context.Context, userID, textVersion string, at time.Time) (domain.Consent, error) {
	var c domain.Consent
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `INSERT INTO consents (user_id, text_version, accepted_at) VALUES ($1, $2, $3)
			ON CONFLICT (user_id, text_version) DO NOTHING`, userID, textVersion, at); err != nil {
			return err
		}
		return tx.QueryRow(ctx, `SELECT text_version, accepted_at FROM consents WHERE user_id = $1 AND text_version = $2`,
			userID, textVersion).Scan(&c.TextVersion, &c.AcceptedAt)
	})
	return c, nf(err)
}

func (s *Store) Consents(ctx context.Context, userID string) ([]domain.Consent, error) {
	rows, err := s.pool.Query(ctx, `SELECT text_version, accepted_at FROM consents WHERE user_id = $1
		ORDER BY accepted_at, text_version`, userID)
	return collect(rows, err, func(r pgx.Row) (c domain.Consent, err error) {
		err = r.Scan(&c.TextVersion, &c.AcceptedAt)
		return
	})
}

// CSV import jobs (D38). import_jobs has no user column; ownership comes from the subject.

const importCols = `j.id::text, j.subject_id::text, j.filename, j.status, j.rows_total, j.rows_imported, j.row_errors, j.created_at, j.finished_at`

func scanImport(r pgx.Row) (j domain.ImportJob, err error) {
	var errs []byte
	if err = r.Scan(&j.ID, &j.SubjectID, &j.Filename, &j.Status, &j.RowsTotal, &j.RowsImported, &errs, &j.CreatedAt, &j.FinishedAt); err != nil {
		return
	}
	err = json.Unmarshal(errs, &j.RowErrors)
	if j.RowErrors == nil {
		j.RowErrors = []domain.ImportRowError{}
	}
	return
}

func (s *Store) CreateImport(ctx context.Context, userID, subjectID, filename string) (domain.ImportJob, error) {
	j, err := scanImport(s.pool.QueryRow(ctx, `WITH j AS (
			INSERT INTO import_jobs (subject_id, filename)
			SELECT id, $3 FROM subjects WHERE id = $1 AND user_id = $2
			RETURNING *)
		SELECT `+importCols+` FROM j`, subjectID, userID, filename))
	return j, nf(err)
}

func (s *Store) GetImport(ctx context.Context, userID, importID string) (domain.ImportJob, error) {
	j, err := scanImport(s.pool.QueryRow(ctx, `SELECT `+importCols+` FROM import_jobs j
		JOIN subjects s ON s.id = j.subject_id WHERE j.id = $1 AND s.user_id = $2`, importID, userID))
	return j, nf(err)
}

func (s *Store) FinishImport(ctx context.Context, userID string, j domain.ImportJob, at time.Time) (domain.ImportJob, error) {
	if j.Status != domain.ImportDone && j.Status != domain.ImportFailed {
		return domain.ImportJob{}, fmt.Errorf("store: finish import with status %q", j.Status)
	}
	errs := j.RowErrors
	if errs == nil {
		errs = []domain.ImportRowError{}
	}
	raw, err := json.Marshal(errs)
	if err != nil {
		return domain.ImportJob{}, err
	}
	out, err := scanImport(s.pool.QueryRow(ctx, `WITH j AS (
			UPDATE import_jobs SET status = $3, rows_total = $4, rows_imported = $5, row_errors = $6, finished_at = $7
			WHERE id = $1 AND subject_id IN (SELECT id FROM subjects WHERE user_id = $2)
			RETURNING *)
		SELECT `+importCols+` FROM j`, j.ID, userID, string(j.Status), j.RowsTotal, j.RowsImported, raw, at))
	return out, nf(err)
}

func (s *Store) ListImports(ctx context.Context, userID, subjectID string) ([]domain.ImportJob, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+importCols+` FROM import_jobs j
		JOIN subjects s ON s.id = j.subject_id WHERE j.subject_id = $1 AND s.user_id = $2
		ORDER BY j.created_at DESC, j.id`, subjectID, userID)
	return collect(rows, err, scanImport)
}

// Disputes into eval cases (D39). Cris and Dino tools only, so not user scoped.

func (s *Store) ResolveDispute(ctx context.Context, disputeID string, outcome domain.DisputeOutcome, expected *domain.ClaimLabel) error {
	switch {
	case outcome == domain.DisputeUpheld && expected == nil:
		return errors.New("store: an upheld dispute needs the expected label")
	case outcome == domain.DisputeRejected && expected != nil:
		return errors.New("store: a rejected dispute has no expected label")
	case outcome != domain.DisputeUpheld && outcome != domain.DisputeRejected:
		return fmt.Errorf("store: unknown dispute outcome %q", outcome)
	}
	var exp *string
	if expected != nil {
		e := string(*expected)
		exp = &e
	}
	return affected(s.pool.Exec(ctx, `UPDATE disputes SET status = 'reviewed', outcome = $2, expected_label = $3 WHERE id = $1`,
		disputeID, string(outcome), exp))
}

// UpheldDisputes joins each dispute to the checker label of the same claim in
// its session (for a dispute with no claim, the session's unmatched label).
// A dispute with no such label is skipped: it has no checker label to compare.
func (s *Store) UpheldDisputes(ctx context.Context) ([]domain.DisputeCase, error) {
	rows, err := s.pool.Query(ctx, `SELECT d.id::text, d.session_id::text, ts.concept_id::text, d.claim_id::text,
			l.student_text, l.label, d.expected_label, d.note
		FROM disputes d
		JOIN teach_sessions ts ON ts.id = d.session_id
		JOIN LATERAL (SELECT student_text, label FROM claim_labels cl
			WHERE cl.session_id = d.session_id AND cl.claim_id IS NOT DISTINCT FROM d.claim_id
			ORDER BY cl.created_at, cl.id LIMIT 1) l ON true
		WHERE d.outcome = 'upheld' ORDER BY d.created_at, d.id`)
	return collect(rows, err, func(r pgx.Row) (c domain.DisputeCase, err error) {
		err = r.Scan(&c.DisputeID, &c.SessionID, &c.ConceptID, &c.ClaimID, &c.StudentText, &c.CheckerLabel, &c.ExpectedLabel, &c.Note)
		return
	})
}

// Metrics (D36). Weeks start Monday 00:00 UTC; Postgres date_trunc('week') is Monday.

func (s *Store) WeekCounts(ctx context.Context, from, to time.Time) ([]domain.WeekCounts, error) {
	rows, err := s.pool.Query(ctx, `
		WITH weeks AS (
			SELECT generate_series($1::timestamptz AT TIME ZONE 'UTC', ($2::timestamptz AT TIME ZONE 'UTC') - interval '1 second', interval '7 days') AS w),
		fin AS (
			SELECT date_trunc('week', ts.finished_at AT TIME ZONE 'UTC') AS w, sub.user_id, ts.id,
				EXISTS (SELECT 1 FROM review_events e WHERE e.session_id = ts.id AND e.event_type = 'teach_back') AS checked
			FROM teach_sessions ts JOIN concepts c ON c.id = ts.concept_id JOIN subjects sub ON sub.id = c.subject_id
			WHERE ts.status = 'finished' AND ts.finished_at IS NOT NULL),
		disp AS (SELECT date_trunc('week', created_at AT TIME ZONE 'UTC') AS w, count(*) AS n FROM disputes GROUP BY 1)
		SELECT (weeks.w AT TIME ZONE 'UTC'),
			(SELECT count(DISTINCT user_id) FROM fin WHERE fin.w = weeks.w)::int,
			(SELECT count(*) FROM fin WHERE fin.w = weeks.w)::int,
			(SELECT count(*) FROM fin WHERE fin.w = weeks.w AND checked)::int,
			(SELECT count(*) FROM fin WHERE fin.w = weeks.w AND NOT checked)::int,
			COALESCE((SELECT n FROM disp WHERE disp.w = weeks.w), 0)::int
		FROM weeks ORDER BY weeks.w`, from, to)
	return collect(rows, err, func(r pgx.Row) (w domain.WeekCounts, err error) {
		err = r.Scan(&w.WeekStart, &w.ActiveUsers, &w.Sessions, &w.CheckedSessions, &w.UncheckedSessions, &w.Disputes)
		return
	})
}

func (s *Store) UserWeeks(ctx context.Context) ([]domain.UserWeeks, error) {
	rows, err := s.pool.Query(ctx, `SELECT u.id::text, u.created_at,
			COALESCE(array_agg(DISTINCT date_trunc('week', ts.finished_at AT TIME ZONE 'UTC') AT TIME ZONE 'UTC')
				FILTER (WHERE ts.id IS NOT NULL), '{}')
		FROM users u
		LEFT JOIN subjects sub ON sub.user_id = u.id
		LEFT JOIN concepts c ON c.subject_id = sub.id
		LEFT JOIN teach_sessions ts ON ts.concept_id = c.id AND ts.status = 'finished' AND ts.finished_at IS NOT NULL
		GROUP BY u.id, u.created_at ORDER BY u.created_at, u.id`)
	return collect(rows, err, func(r pgx.Row) (u domain.UserWeeks, err error) {
		err = r.Scan(&u.UserID, &u.JoinedAt, &u.ActiveWeeks)
		if u.ActiveWeeks == nil {
			u.ActiveWeeks = []time.Time{}
		}
		return
	})
}

// Spend ledger (D33). One row per UTC month; Add is a single atomic upsert.

func monthOf(at time.Time) time.Time {
	y, m, _ := at.UTC().Date()
	return time.Date(y, m, 1, 0, 0, 0, 0, time.UTC)
}

func (s *Store) Add(ctx context.Context, at time.Time, usd float64) (float64, error) {
	if usd < 0 || math.IsNaN(usd) || math.IsInf(usd, 0) {
		return 0, fmt.Errorf("store: spend must be a finite number >= 0, got %v", usd)
	}
	var total float64
	err := s.pool.QueryRow(ctx, `INSERT INTO model_spend (month, usd) VALUES ($1, $2)
		ON CONFLICT (month) DO UPDATE SET usd = model_spend.usd + EXCLUDED.usd, updated_at = now()
		RETURNING usd::float8`, monthOf(at), usd).Scan(&total)
	return total, err
}

func (s *Store) MonthTotal(ctx context.Context, at time.Time) (float64, error) {
	var total float64
	err := s.pool.QueryRow(ctx, `SELECT COALESCE((SELECT usd::float8 FROM model_spend WHERE month = $1), 0)`, monthOf(at)).Scan(&total)
	return total, err
}
