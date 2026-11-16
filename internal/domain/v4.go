package domain

import (
	"context"
	"time"
)

// Contract v4 (Phase 2, D36 to D44). Pilot tooling: feedback, consent,
// privacy export and account deletion, CSV import, disputes into eval cases,
// metrics from existing rows and a persistent model spend ledger.

// Feedback (D40). One optional question per week; answers are never shown to others.

// FeedbackAnswer is one user's answer for one week (local Monday, as a date).
type FeedbackAnswer struct {
	WeekStart time.Time `json:"week_start"`
	Question  string    `json:"question"`
	Answer    string    `json:"answer"`
	CreatedAt time.Time `json:"created_at"`
}

// FeedbackPrompt is this week's question, chosen by code, and the user's
// answer if they gave one.
type FeedbackPrompt struct {
	WeekStart time.Time `json:"week_start"`
	Question  string    `json:"question"`
	Answer    *string   `json:"answer"`
}

// Consent records which version of the plain-words text the user accepted (D44).
type Consent struct {
	TextVersion string    `json:"text_version"`
	AcceptedAt  time.Time `json:"accepted_at"`
}

// CSV import (D38).

type ImportStatus string

const (
	ImportRunning ImportStatus = "running"
	ImportDone    ImportStatus = "done"   // finished; some rows may still be in RowErrors
	ImportFailed  ImportStatus = "failed" // the file itself could not be read
)

// ImportRow is one CSV line: concept, claim, kind, quote (D38).
type ImportRow struct {
	Concept string    `json:"concept"`
	Claim   string    `json:"claim"`
	Kind    ClaimKind `json:"kind"`
	Quote   string    `json:"quote"`
}

// ImportRowError is a refused row. Row counts from 1 at the header line, so it
// matches the line number a spreadsheet shows. Error is plain words.
type ImportRowError struct {
	Row   int    `json:"row"`
	Error string `json:"error"`
}

type ImportJob struct {
	ID           string           `json:"id"`
	SubjectID    string           `json:"subject_id"`
	Filename     string           `json:"filename"`
	Status       ImportStatus     `json:"status"`
	RowsTotal    int              `json:"rows_total"`
	RowsImported int              `json:"rows_imported"`
	RowErrors    []ImportRowError `json:"row_errors"`
	CreatedAt    time.Time        `json:"created_at"`
	FinishedAt   *time.Time       `json:"finished_at"`
}

// Privacy export (D37).

// TeachTurn is one turn of a teach session as stored.
type TeachTurn struct {
	SessionID     string    `json:"session_id"`
	TurnNo        int       `json:"turn_no"`
	Transcript    string    `json:"transcript"`
	Latex         string    `json:"latex"`
	AssistantText string    `json:"assistant_text"`
	CreatedAt     time.Time `json:"created_at"`
}

// UserExport is "Download my data": everything the user wrote and every
// schedule change. Upload file bytes and page images are left out.
type UserExport struct {
	ExportedAt   time.Time        `json:"exported_at"`
	User         User             `json:"user"`
	Consents     []Consent        `json:"consents"`
	Feedback     []FeedbackAnswer `json:"feedback"`
	Courses      []Subject        `json:"courses"`
	Concepts     []Concept        `json:"concepts"`
	Uploads      []Upload         `json:"uploads"`
	Pages        []Page           `json:"pages"`
	Sessions     []TeachSession   `json:"sessions"`
	Turns        []TeachTurn      `json:"turns"`
	TaughtClaims []TaughtClaim    `json:"taught_claims"`
	Reports      []Report         `json:"reports"`
	SelfGrades   []SelfGrade      `json:"self_grades"`
	Disputes     []Dispute        `json:"disputes"`
	ReviewEvents []ReviewEvent    `json:"review_events"`
	Imports      []ImportJob      `json:"imports"`
}

// Disputes into eval cases (D39).

type DisputeOutcome string

const (
	DisputeUpheld   DisputeOutcome = "upheld"
	DisputeRejected DisputeOutcome = "rejected"
)

// DisputeCase is an upheld dispute shaped as a dev-split eval case: what the
// student said, what the checker labelled, and what a person says is right.
type DisputeCase struct {
	DisputeID     string     `json:"dispute_id"`
	SessionID     string     `json:"session_id"`
	ConceptID     string     `json:"concept_id"`
	ClaimID       *string    `json:"claim_id"`
	StudentText   string     `json:"student_text"`
	CheckerLabel  ClaimLabel `json:"checker_label"`
	ExpectedLabel ClaimLabel `json:"expected_label"`
	Note          string     `json:"note"`
}

// Pilot metrics (D36). Read-only counts over existing rows, no tracking.

// WeekCounts covers one week starting WeekStart (Monday 00:00 UTC).
type WeekCounts struct {
	WeekStart         time.Time `json:"week_start"`
	ActiveUsers       int       `json:"active_users"`       // users with at least one finished session
	Sessions          int       `json:"sessions"`           // finished teach sessions
	CheckedSessions   int       `json:"checked_sessions"`   // sessions with a teach_back event
	UncheckedSessions int       `json:"unchecked_sessions"` // finished without one
	Disputes          int       `json:"disputes"`
}

// UserWeeks is one user's join time and the weeks (Monday UTC) they finished a
// session, for week-4 retention. UserID is internal; exports for people other
// than Cris must not carry it.
type UserWeeks struct {
	UserID      string      `json:"user_id"`
	JoinedAt    time.Time   `json:"joined_at"`
	ActiveWeeks []time.Time `json:"active_weeks"`
}

// Interfaces for wave 2. Store methods take the signed-in user's id and return
// ErrNotFound for rows of other users, as in v3.

// FeedbackStore holds weekly answers and consent records.
type FeedbackStore interface {
	// SaveFeedback stores or replaces the user's answer for a.WeekStart.
	SaveFeedback(ctx context.Context, userID string, a FeedbackAnswer) (FeedbackAnswer, error)
	// GetFeedback returns ErrNotFound when the user has not answered that week.
	GetFeedback(ctx context.Context, userID string, weekStart time.Time) (FeedbackAnswer, error)
	// SaveConsent is idempotent per text version: the first accepted_at stays.
	SaveConsent(ctx context.Context, userID, textVersion string, at time.Time) (Consent, error)
	Consents(ctx context.Context, userID string) ([]Consent, error)
}

// PrivacyStore is "Download my data" and "Delete my account" (D37, D43).
type PrivacyStore interface {
	ExportUser(ctx context.Context, userID string, now time.Time) (UserExport, error)
	// DeleteUser removes every row of the user in one transaction, review_events
	// included. It is the only code that may run SET LOCAL app.delete_user (D43).
	DeleteUser(ctx context.Context, userID string) error
}

// ImportStore persists import jobs.
type ImportStore interface {
	CreateImport(ctx context.Context, userID, subjectID, filename string) (ImportJob, error)
	GetImport(ctx context.Context, userID, importID string) (ImportJob, error)
	// FinishImport sets the final status, counts and row errors, and finished_at.
	FinishImport(ctx context.Context, userID string, j ImportJob, at time.Time) (ImportJob, error)
	ListImports(ctx context.Context, userID, subjectID string) ([]ImportJob, error)
}

// Importer reads a CSV of ImportRow lines into a course. Every claim goes
// through CourseNotes.AddClaim, so the quote check is the same as for a typed
// claim (D28, D38). It returns the job; the work may finish later.
type Importer interface {
	Import(ctx context.Context, userID, subjectID, filename string, csv []byte) (ImportJob, error)
}

// EvalCaseStore turns disputes into dev-split eval cases (D39). Used by cmd
// tools for Dino and Cris, never by the student API.
type EvalCaseStore interface {
	// ResolveDispute marks the dispute reviewed with an outcome. expected is
	// required for upheld and must be nil for rejected.
	ResolveDispute(ctx context.Context, disputeID string, outcome DisputeOutcome, expected *ClaimLabel) error
	// UpheldDisputes returns every upheld dispute as a case, oldest first.
	UpheldDisputes(ctx context.Context) ([]DisputeCase, error)
}

// MetricsStore is read-only over existing rows (D36). For cmd/export, all users.
type MetricsStore interface {
	// WeekCounts returns one row per week in [from, to), both Mondays UTC.
	WeekCounts(ctx context.Context, from, to time.Time) ([]WeekCounts, error)
	UserWeeks(ctx context.Context) ([]UserWeeks, error)
}

// SpendLedger keeps model spend per calendar month (UTC) in Postgres, so the
// D33 monthly cap survives restarts. P2-API wires it into the cap.
type SpendLedger interface {
	// Add adds usd (>= 0) to the month of at and returns that month's new total.
	Add(ctx context.Context, at time.Time, usd float64) (float64, error)
	// MonthTotal returns the month's total, 0 for a month with no spend.
	MonthTotal(ctx context.Context, at time.Time) (float64, error)
}
