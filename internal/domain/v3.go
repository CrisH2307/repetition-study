package domain

import (
	"context"
	"errors"
	"time"
)

// Contract v3 (Phase 1, D26 to D34). Course is the product word for Subject.

// ErrNotFound means the row does not exist or belongs to another user. The two
// cases look the same on purpose, so ids of other users leak nothing (404).
var ErrNotFound = errors.New("not found")

// UserError is a refusal the student can act on. Msg is plain words and is
// shown as is (400), for example "That quote is not on a Ready page."
type UserError struct{ Msg string }

func (e *UserError) Error() string { return e.Msg }

// Users and sign-in (D29).

type User struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	TimeZone string `json:"time_zone"` // IANA name
	// IsInstructor is set by cmd/instructor only (D51, contract v5).
	IsInstructor bool `json:"is_instructor"`
}

type Invite struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	CreatedAt time.Time  `json:"created_at"`
	UsedAt    *time.Time `json:"used_at"`
	UserID    *string    `json:"user_id"`
}

// Uploads and pages (D27).

type UploadStatus string

const (
	UploadReading       UploadStatus = "reading"
	UploadNeedsChecking UploadStatus = "needs_checking"
	UploadReady         UploadStatus = "ready"
)

type PageStatus string

const (
	PageNeedsChecking PageStatus = "needs_checking"
	PageReady         PageStatus = "ready"
)

// Upload is one PDF or pasted text in a course. The file bytes are not part of
// the JSON. Error is plain words when reading failed.
type Upload struct {
	ID         string       `json:"id"`
	SubjectID  string       `json:"subject_id"`
	Filename   string       `json:"filename"`
	Status     UploadStatus `json:"status"`
	Error      string       `json:"error"`
	SourceID   *string      `json:"source_id"`
	PagesTotal int          `json:"pages_total"`
	PagesReady int          `json:"pages_ready"`
	CreatedAt  time.Time    `json:"created_at"`
}

// Page is one page of an upload. RawText is what code read; EditedText is the
// student's checked version and the only text that becomes source blocks.
// The PNG is served by its own endpoint.
type Page struct {
	ID         string     `json:"id"`
	UploadID   string     `json:"upload_id"`
	Number     int        `json:"number"`
	RawText    string     `json:"raw_text"`
	EditedText string     `json:"edited_text"`
	Status     PageStatus `json:"status"`
	HasImage   bool       `json:"has_image"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// Claim checklist (D28). Claim gains Kind, AuthorUserID and Anchor (v3 fields in types.go).

type ClaimKind string

const (
	KindDefinition ClaimKind = "definition"
	KindStatement  ClaimKind = "statement"
	KindCondition  ClaimKind = "condition"
	KindMethod     ClaimKind = "method"
)

// Anchor ties a claim to an exact quote in one Ready source block. internal/cite verifies it.
type Anchor struct {
	BlockID string `json:"block_id"`
	Quote   string `json:"quote"`
}

// ClaimInput is what the student sends to add or edit a claim.
type ClaimInput struct {
	Text     string    `json:"text"`
	Kind     ClaimKind `json:"kind"`
	Required bool      `json:"required"`
	Anchor   Anchor    `json:"anchor"`
}

// Self-grades and disputes (D30). Never change a schedule.

type SelfRating string

const (
	RatingRight   SelfRating = "right"
	RatingNotSure SelfRating = "not_sure"
	RatingWrong   SelfRating = "wrong"
)

type SelfGrade struct {
	SessionID string     `json:"session_id"`
	ClaimID   string     `json:"claim_id"`
	Rating    SelfRating `json:"rating"`
}

type DisputeStatus string

const (
	DisputeOpen     DisputeStatus = "open"
	DisputeReviewed DisputeStatus = "reviewed"
)

type Dispute struct {
	ID        string        `json:"id"`
	SessionID string        `json:"session_id"`
	ClaimID   *string       `json:"claim_id"`
	UserID    string        `json:"user_id"`
	Note      string        `json:"note"`
	Status    DisputeStatus `json:"status"`
	CreatedAt time.Time     `json:"created_at"`
}

// Today and week (D31). Every number carries a reason in words.

type TodayKind string

const (
	TodayTeach  TodayKind = "teach"  // 15 minutes
	TodayRecall TodayKind = "recall" // 5 minutes
)

type TodayItem struct {
	ConceptID   string    `json:"concept_id"`
	ConceptName string    `json:"concept_name"`
	SubjectID   string    `json:"subject_id"`
	Kind        TodayKind `json:"kind"`
	Minutes     int       `json:"minutes"`
	DueAt       time.Time `json:"due_at"`
	Reason      string    `json:"reason"`
}

// CourseHours is "1.2 / 3.5 h". Reason is empty unless the budget changed.
type CourseHours struct {
	SubjectID   string  `json:"subject_id"`
	Name        string  `json:"name"`
	HoursUsed   float64 `json:"hours_used"`
	HoursBudget float64 `json:"hours_budget"`
	Reason      string  `json:"reason"`
}

type TodayCourse struct {
	SubjectID string      `json:"subject_id"`
	Name      string      `json:"name"`
	Minutes   int         `json:"minutes"`
	Items     []TodayItem `json:"items"`
}

type Today struct {
	Date      time.Time     `json:"date"` // local midnight of the user's day, as UTC
	Courses   []TodayCourse `json:"courses"`
	StudyTime []CourseHours `json:"study_time"`
}

type WeekDay struct {
	Date  time.Time   `json:"date"`
	Items []TodayItem `json:"items"`
}

type WeekPlan struct {
	WeekStart time.Time     `json:"week_start"`
	Courses   []CourseHours `json:"courses"`
	Days      []WeekDay     `json:"days"`
	Notes     []string      `json:"notes"`
}

// StudyState is everything the Today and week code reads for one user. The
// store builds it; the planner is pure code over it.
type StudyState struct {
	User    User          `json:"user"`
	Courses []CourseState `json:"courses"`
}

type CourseState struct {
	Subject  Subject        `json:"subject"`
	Concepts []ConceptState `json:"concepts"`
	// Events since the start of the week asked for, oldest first.
	Events []ReviewEvent `json:"events"`
}

type ConceptState struct {
	ConceptID string     `json:"concept_id"`
	Name      string     `json:"name"`
	DueAt     *time.Time `json:"due_at"` // nil: never taught, counts as due
	HasCards  bool       `json:"has_cards"`
	Taught    bool       `json:"taught"` // at least one checked teach-back
}

// Course view (screen 6).

type StudyCard struct {
	ConceptID   string     `json:"concept_id"`
	Name        string     `json:"name"`
	Recall      *float64   `json:"recall"`
	Explanation *float64   `json:"explanation"`
	NextDue     *time.Time `json:"next_due"`
	Claims      []Claim    `json:"claims"`
}

// Mistake is one contradicted or partial claim from a checked report, linked
// to its passage.
type Mistake struct {
	SessionID   string     `json:"session_id"`
	ConceptID   string     `json:"concept_id"`
	ClaimID     *string    `json:"claim_id"`
	StudentText string     `json:"student_text"`
	Label       ClaimLabel `json:"label"`
	Quote       string     `json:"quote"`
	SourceID    string     `json:"source_id"`
	PageID      *string    `json:"page_id"`
	Page        *int       `json:"page"`
	CreatedAt   time.Time  `json:"created_at"`
}

type CourseView struct {
	Course   Subject     `json:"course"`
	Notes    []Upload    `json:"notes"`
	Cards    []StudyCard `json:"cards"`
	Mistakes []Mistake   `json:"mistakes"`
}

// Interfaces for wave 2. Every store method takes the signed-in user's id and
// returns ErrNotFound for rows of other users.

// AuthStore persists invites and sign-in sessions. P1-STORE implements it.
// Hashes are SHA-256 of the random token; tokens never reach the store.
type AuthStore interface {
	CreateInvite(ctx context.Context, name string, tokenHash []byte) (Invite, error)
	// RedeemInvite creates the invite's user on first use and returns that
	// same user after. ErrNotFound for an unknown hash.
	RedeemInvite(ctx context.Context, tokenHash []byte, now time.Time) (User, error)
	CreateSession(ctx context.Context, userID string, tokenHash []byte, expiresAt time.Time) error
	// SessionUser returns ErrNotFound for an unknown or expired session.
	SessionUser(ctx context.Context, tokenHash []byte, now time.Time) (User, error)
	DeleteSession(ctx context.Context, tokenHash []byte) error
}

// Auth is sign-in by invite link. P1-AUTH implements it; the HTTP middleware
// and cookie live in internal/auth.
type Auth interface {
	// NewInvite returns the one-time token for the link. Only its hash is stored.
	NewInvite(ctx context.Context, name string) (token string, err error)
	// Redeem returns the user and a new session token valid until expires.
	Redeem(ctx context.Context, inviteToken string) (u User, sessionToken string, expires time.Time, err error)
	// UserFor returns ErrNotFound for a missing, tampered or expired session.
	UserFor(ctx context.Context, sessionToken string) (User, error)
	SignOut(ctx context.Context, sessionToken string) error
}

// CourseStore holds courses, concepts, uploads, pages, blocks and claims.
// P1-STORE implements it.
type CourseStore interface {
	ListCourses(ctx context.Context, userID string) ([]Subject, error)
	GetCourse(ctx context.Context, userID, subjectID string) (Subject, error)
	CreateCourse(ctx context.Context, userID string, s Subject) (Subject, error)
	UpdateCourse(ctx context.Context, userID string, s Subject) (Subject, error)
	// ArchiveCourse hides the course. Rows stay, review_events is append-only.
	ArchiveCourse(ctx context.Context, userID, subjectID string) error
	CreateConcept(ctx context.Context, userID, subjectID, name string) (Concept, error)
	// GetConcept returns the concept with its live (not deleted) claims.
	GetConcept(ctx context.Context, userID, conceptID string) (Concept, error)

	// CreateUpload stores the PDF (at most 20 MB) or the pasted text, status reading.
	CreateUpload(ctx context.Context, userID string, u Upload, pdf []byte, pastedText string) (Upload, error)
	GetUpload(ctx context.Context, userID, uploadID string) (Upload, error)
	ListUploads(ctx context.Context, userID, subjectID string) ([]Upload, error)
	// UploadFile returns the stored PDF bytes and pasted text (one is empty).
	UploadFile(ctx context.Context, userID, uploadID string) (pdf []byte, pastedText string, err error)
	SetUploadStatus(ctx context.Context, userID, uploadID string, status UploadStatus, errMsg string) error
	// SavePages writes the pages read from an upload. images[i] is pages[i]'s PNG, or nil.
	SavePages(ctx context.Context, userID, uploadID string, pages []Page, images [][]byte) error
	ListPages(ctx context.Context, userID, uploadID string) ([]Page, error)
	GetPage(ctx context.Context, userID, pageID string) (Page, error)
	PageImage(ctx context.Context, userID, pageID string) ([]byte, error)
	EditPage(ctx context.Context, userID, pageID, editedText string) (Page, error)
	// SetPageReady marks the page Ready and replaces its source blocks with
	// blocks, under the upload's course source (created on first use).
	SetPageReady(ctx context.Context, userID, pageID string, blocks []SourceBlock) (Page, error)
	// SetPageNotReady removes the page's blocks. A UserError if a live claim is anchored to one.
	SetPageNotReady(ctx context.Context, userID, pageID string) (Page, error)
	// ReadyBlocks returns the course's blocks from Ready pages: the only quotable text.
	ReadyBlocks(ctx context.Context, userID, subjectID string) ([]SourceBlock, error)
	// CourseSources returns the approved sources of the course, for the checker.
	CourseSources(ctx context.Context, userID, subjectID string) ([]Source, error)

	ListClaims(ctx context.Context, userID, conceptID string) ([]Claim, error)
	// CreateClaim and UpdateClaim store an anchor the caller already verified.
	CreateClaim(ctx context.Context, userID, conceptID string, in ClaimInput) (Claim, error)
	UpdateClaim(ctx context.Context, userID, claimID string, in ClaimInput) (Claim, error)
	DeleteClaim(ctx context.Context, userID, claimID string) error
}

// ReviewStore holds self-grades, disputes and the read models built from
// events. P1-STORE implements it. Nothing here writes review_events.
type ReviewStore interface {
	// SaveSelfGrades replaces the session's self-grades.
	SaveSelfGrades(ctx context.Context, userID, sessionID string, grades []SelfGrade) error
	CreateDispute(ctx context.Context, userID string, d Dispute) (Dispute, error)
	// AllDisputes is for cmd/export only: every user, oldest first.
	AllDisputes(ctx context.Context) ([]Dispute, error)
	CourseView(ctx context.Context, userID, subjectID string) (CourseView, error)
	// StudyState loads the user's courses with events since since.
	StudyState(ctx context.Context, userID string, since time.Time) (StudyState, error)
	// SessionOwner returns ErrNotFound unless the teach session is the user's.
	SessionOwner(ctx context.Context, userID, sessionID string) error
}

// CourseNotes is the course logic: reading uploads into pages, the page check
// and the claim checklist. P1-COURSE implements it over CourseStore and internal/cite.
type CourseNotes interface {
	// ReadUpload splits the upload into pages and sets its status. A bad file
	// sets status needs_checking with no pages and a plain Error, never panics.
	ReadUpload(ctx context.Context, userID, uploadID string) error
	EditPage(ctx context.Context, userID, pageID, editedText string) (Page, error)
	MarkReady(ctx context.Context, userID, pageID string) (Page, error)
	MarkNotReady(ctx context.Context, userID, pageID string) (Page, error)
	// AddClaim and EditClaim refuse with a UserError when the anchor quote is
	// not on a Ready page of the claim's course.
	AddClaim(ctx context.Context, userID, conceptID string, in ClaimInput) (Claim, error)
	EditClaim(ctx context.Context, userID, claimID string, in ClaimInput) (Claim, error)
	DeleteClaim(ctx context.Context, userID, claimID string) error
}

// TodayPlanner is pure code over StudyState (D31). P1-TODAY implements it.
type TodayPlanner interface {
	Today(st StudyState, now time.Time) (Today, error)
	// Week plans the week starting weekStart (local Monday, as UTC).
	Week(st StudyState, weekStart, now time.Time) (WeekPlan, error)
}
