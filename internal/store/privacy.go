package store

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/CrisH2307/repetition-study/internal/domain"
)

var _ domain.PrivacyStore = (*Store)(nil)

// Ownership fragments. $1 is always the user id. Every table reaches its user
// through subjects (or carries user_id), so export and delete use the same
// scoping and cannot drift apart.
const (
	ownSubjects = `(SELECT id FROM subjects WHERE user_id = $1)`
	ownConcepts = `(SELECT id FROM concepts WHERE subject_id IN ` + ownSubjects + `)`
	ownSessions = `(SELECT id FROM teach_sessions WHERE concept_id IN ` + ownConcepts + `)`
	ownUploads  = `(SELECT id FROM uploads WHERE subject_id IN ` + ownSubjects + `)`
	ownSources  = `(SELECT id FROM sources WHERE subject_id IN ` + ownSubjects + `)`
)

// ExportUser returns everything the user owns, read in one snapshot. Upload
// file bytes and page images are left out.
func (s *Store) ExportUser(ctx context.Context, userID string, now time.Time) (domain.UserExport, error) {
	out := domain.UserExport{ExportedAt: now}
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `SET TRANSACTION ISOLATION LEVEL REPEATABLE READ, READ ONLY`); err != nil {
			return err
		}
		if err := tx.QueryRow(ctx, `SELECT id::text, name, time_zone FROM users WHERE id = $1`, userID).
			Scan(&out.User.ID, &out.User.Name, &out.User.TimeZone); err != nil {
			return nf(err)
		}
		var err error
		list := func(sql string, dst func(pgx.Rows) error) {
			if err != nil {
				return
			}
			var rows pgx.Rows
			rows, err = tx.Query(ctx, sql, userID)
			if err != nil {
				return
			}
			defer rows.Close()
			for rows.Next() {
				if err = dst(rows); err != nil {
					return
				}
			}
			err = rows.Err()
		}
		list(`SELECT text_version, accepted_at FROM consents WHERE user_id = $1 ORDER BY accepted_at, text_version`, func(r pgx.Rows) error {
			var x domain.Consent
			err := r.Scan(&x.TextVersion, &x.AcceptedAt)
			out.Consents = append(out.Consents, x)
			return err
		})
		list(`SELECT week_start, question, answer, created_at FROM feedback_answers WHERE user_id = $1 ORDER BY week_start`, func(r pgx.Rows) error {
			var x domain.FeedbackAnswer
			err := r.Scan(&x.WeekStart, &x.Question, &x.Answer, &x.CreatedAt)
			x.WeekStart = x.WeekStart.UTC()
			out.Feedback = append(out.Feedback, x)
			return err
		})
		list(`SELECT `+subjectCols+` FROM subjects s WHERE s.user_id = $1 ORDER BY s.created_at, s.id`, func(r pgx.Rows) error {
			x, err := scanSubject(r)
			out.Courses = append(out.Courses, x)
			return err
		})
		list(`SELECT id::text, subject_id::text, name FROM concepts WHERE subject_id IN `+ownSubjects+` ORDER BY created_at, id`, func(r pgx.Rows) error {
			x := domain.Concept{Claims: []domain.Claim{}}
			err := r.Scan(&x.ID, &x.SubjectID, &x.Name)
			out.Concepts = append(out.Concepts, x)
			return err
		})
		// Hidden (deleted) claims are the user's words too, so they are included.
		var claims []domain.Claim
		list(`SELECT `+claimCols+` FROM claims c WHERE c.concept_id IN `+ownConcepts+` ORDER BY c.created_at, c.id`, func(r pgx.Rows) error {
			x, err := scanClaim(r)
			claims = append(claims, x)
			return err
		})
		list(`SELECT u.id::text, u.subject_id::text, u.filename, u.status, u.error, u.source_id::text,
				count(p.id)::int, (count(p.id) FILTER (WHERE p.status = 'ready'))::int, u.created_at
			FROM uploads u LEFT JOIN pages p ON p.upload_id = u.id
			WHERE u.subject_id IN `+ownSubjects+` GROUP BY u.id ORDER BY u.created_at, u.id`, func(r pgx.Rows) error {
			x, err := scanUpload(r)
			out.Uploads = append(out.Uploads, x)
			return err
		})
		list(`SELECT p.id::text, p.upload_id::text, p.number, p.raw_text, p.edited_text, p.status, p.image_png IS NOT NULL, p.updated_at
			FROM pages p WHERE p.upload_id IN `+ownUploads+` ORDER BY p.upload_id, p.number`, func(r pgx.Rows) error {
			x, err := scanPage(r)
			out.Pages = append(out.Pages, x)
			return err
		})
		list(`SELECT id::text, concept_id::text FROM teach_sessions WHERE concept_id IN `+ownConcepts+` ORDER BY started_at, id`, func(r pgx.Rows) error {
			var x domain.TeachSession
			err := r.Scan(&x.ID, &x.ConceptID)
			out.Sessions = append(out.Sessions, x)
			return err
		})
		list(`SELECT session_id::text, turn_no, transcript, latex, assistant_text, created_at
			FROM teach_turns WHERE session_id IN `+ownSessions+` ORDER BY session_id, turn_no`, func(r pgx.Rows) error {
			var x domain.TeachTurn
			err := r.Scan(&x.SessionID, &x.TurnNo, &x.Transcript, &x.Latex, &x.AssistantText, &x.CreatedAt)
			out.Turns = append(out.Turns, x)
			return err
		})
		list(`SELECT id::text, session_id::text, concept_id::text, student_text, turn_no
			FROM taught_claims WHERE session_id IN `+ownSessions+` ORDER BY session_id, turn_no, id`, func(r pgx.Rows) error {
			var x domain.TaughtClaim
			err := r.Scan(&x.ID, &x.SessionID, &x.ConceptID, &x.StudentText, &x.TurnNo)
			out.TaughtClaims = append(out.TaughtClaims, x)
			return err
		})
		// One report per session that has labels.
		reports := map[string]int{}
		list(`SELECT l.session_id::text, t.concept_id::text, l.claim_id::text, l.student_text, l.label,
				l.citation_source_id::text, l.citation_quote, l.citation_locator, l.correction, l.probe_hint
			FROM claim_labels l JOIN teach_sessions t ON t.id = l.session_id
			WHERE l.session_id IN `+ownSessions+` ORDER BY l.session_id, l.created_at, l.id`, func(r pgx.Rows) error {
			var sid, cid string
			var x domain.ClaimResult
			var src, quote, loc *string
			if err := r.Scan(&sid, &cid, &x.ClaimID, &x.StudentText, &x.Label, &src, &quote, &loc, &x.Correction, &x.ProbeHint); err != nil {
				return err
			}
			if src != nil && quote != nil {
				c := domain.Citation{SourceID: *src, Quote: *quote}
				if loc != nil {
					c.Locator = *loc
				}
				x.Citation = &c
			}
			i, ok := reports[sid]
			if !ok {
				i = len(out.Reports)
				reports[sid] = i
				out.Reports = append(out.Reports, domain.Report{SessionID: sid, ConceptID: cid, Status: domain.ReportChecked})
			}
			out.Reports[i].Results = append(out.Reports[i].Results, x)
			return nil
		})
		list(`SELECT session_id::text, claim_id::text, rating FROM self_grades WHERE user_id = $1 ORDER BY created_at, id`, func(r pgx.Rows) error {
			var x domain.SelfGrade
			err := r.Scan(&x.SessionID, &x.ClaimID, &x.Rating)
			out.SelfGrades = append(out.SelfGrades, x)
			return err
		})
		list(`SELECT `+disputeCols+` FROM disputes WHERE user_id = $1 ORDER BY created_at, id`, func(r pgx.Rows) error {
			x, err := scanDispute(r)
			out.Disputes = append(out.Disputes, x)
			return err
		})
		list(`SELECT id::text, event_type, concept_id::text, card_id::text, grade, due_before, due_after,
				stability_before, stability_after, payload, created_at, session_id::text, claim_id::text,
				self_grade, checker_model, checker_version
			FROM review_events WHERE concept_id IN `+ownConcepts+` ORDER BY created_at, id`, func(r pgx.Rows) error {
			var x domain.ReviewEvent
			err := r.Scan(&x.ID, &x.EventType, &x.ConceptID, &x.CardID, &x.Grade, &x.DueBefore, &x.DueAfter,
				&x.StabilityBefore, &x.StabilityAfter, &x.Payload, &x.CreatedAt, &x.SessionID, &x.ClaimID,
				&x.SelfGrade, &x.CheckerModel, &x.CheckerVersion)
			out.ReviewEvents = append(out.ReviewEvents, x)
			return err
		})
		var jobs []domain.ImportJob
		list(`SELECT id::text, subject_id::text, filename, status, rows_total, rows_imported, row_errors, created_at, finished_at
			FROM import_jobs WHERE subject_id IN `+ownSubjects+` ORDER BY created_at, id`, func(r pgx.Rows) error {
			var x domain.ImportJob
			err := r.Scan(&x.ID, &x.SubjectID, &x.Filename, &x.Status, &x.RowsTotal, &x.RowsImported, &x.RowErrors, &x.CreatedAt, &x.FinishedAt)
			if x.RowErrors == nil {
				x.RowErrors = []domain.ImportRowError{}
			}
			jobs = append(jobs, x)
			return err
		})
		if err != nil {
			return err
		}
		idx := map[string]int{}
		for i, c := range out.Concepts {
			idx[c.ID] = i
		}
		for _, c := range claims {
			if i, ok := idx[c.ConceptID]; ok {
				out.Concepts[i].Claims = append(out.Concepts[i].Claims, c)
			}
		}
		out.Imports = jobs
		return nil
	})
	if err != nil {
		return domain.UserExport{}, err
	}
	// Empty lists encode as [], never null.
	if out.Consents == nil {
		out.Consents = []domain.Consent{}
	}
	if out.Feedback == nil {
		out.Feedback = []domain.FeedbackAnswer{}
	}
	if out.Courses == nil {
		out.Courses = []domain.Subject{}
	}
	if out.Concepts == nil {
		out.Concepts = []domain.Concept{}
	}
	if out.Uploads == nil {
		out.Uploads = []domain.Upload{}
	}
	if out.Pages == nil {
		out.Pages = []domain.Page{}
	}
	if out.Sessions == nil {
		out.Sessions = []domain.TeachSession{}
	}
	if out.Turns == nil {
		out.Turns = []domain.TeachTurn{}
	}
	if out.TaughtClaims == nil {
		out.TaughtClaims = []domain.TaughtClaim{}
	}
	if out.Reports == nil {
		out.Reports = []domain.Report{}
	}
	if out.SelfGrades == nil {
		out.SelfGrades = []domain.SelfGrade{}
	}
	if out.Disputes == nil {
		out.Disputes = []domain.Dispute{}
	}
	if out.ReviewEvents == nil {
		out.ReviewEvents = []domain.ReviewEvent{}
	}
	if out.Imports == nil {
		out.Imports = []domain.ImportJob{}
	}
	return out, nil
}

// deleteSteps run in this order (children before parents). Each takes only the
// user id and reaches rows through ownership, so nothing of another user can
// match. review_events goes first among the course rows: the trigger from
// migration 0005 allows it only while app.delete_user names this user.
var deleteSteps = []string{
	`DELETE FROM disputes WHERE user_id = $1 OR session_id IN ` + ownSessions,
	`DELETE FROM self_grades WHERE user_id = $1 OR session_id IN ` + ownSessions,
	`DELETE FROM review_events WHERE concept_id IN ` + ownConcepts,
	`DELETE FROM claim_labels WHERE session_id IN ` + ownSessions,
	`DELETE FROM taught_claims WHERE session_id IN ` + ownSessions + ` OR concept_id IN ` + ownConcepts,
	`DELETE FROM teach_turns WHERE session_id IN ` + ownSessions,
	`DELETE FROM teach_sessions WHERE concept_id IN ` + ownConcepts,
	`DELETE FROM cards WHERE concept_id IN ` + ownConcepts,
	`DELETE FROM concept_schedules WHERE concept_id IN ` + ownConcepts,
	`DELETE FROM plan_blocks WHERE subject_id IN ` + ownSubjects,
	`DELETE FROM claims WHERE concept_id IN ` + ownConcepts,
	`DELETE FROM import_jobs WHERE subject_id IN ` + ownSubjects,
	`DELETE FROM source_blocks WHERE source_id IN ` + ownSources + ` OR page_id IN (SELECT id FROM pages WHERE upload_id IN ` + ownUploads + `)`,
	`DELETE FROM pages WHERE upload_id IN ` + ownUploads,
	`DELETE FROM uploads WHERE subject_id IN ` + ownSubjects,
	`DELETE FROM sources WHERE subject_id IN ` + ownSubjects,
	`DELETE FROM concepts WHERE subject_id IN ` + ownSubjects,
	`DELETE FROM subjects WHERE user_id = $1`,
	`DELETE FROM feedback_answers WHERE user_id = $1`,
	`DELETE FROM consents WHERE user_id = $1`,
	`DELETE FROM digests WHERE user_id = $1`,
	`DELETE FROM auth_sessions WHERE user_id = $1`,
	`DELETE FROM invites WHERE user_id = $1`,
}

// DeleteUser removes every row of the user in one transaction, review history
// included (D43). It is the only code that sets app.delete_user, and it sets it
// with set_config(..., true), so the setting ends with the transaction. A
// failed step rolls everything back. An unknown or already deleted user is
// ErrNotFound.
func (s *Store) DeleteUser(ctx context.Context, userID string) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		var id string
		if err := tx.QueryRow(ctx, `SELECT id::text FROM users WHERE id = $1 FOR UPDATE`, userID).Scan(&id); err != nil {
			return nf(err)
		}
		if _, err := tx.Exec(ctx, `SELECT set_config('app.delete_user', $1, true)`, id); err != nil {
			return err
		}
		for _, sql := range deleteSteps {
			if _, err := tx.Exec(ctx, sql, id); err != nil {
				return err
			}
		}
		return affected(tx.Exec(ctx, `DELETE FROM users WHERE id = $1`, id))
	})
}
