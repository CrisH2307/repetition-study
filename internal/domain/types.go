// Package domain holds the shared contract. Units import it and never edit it.
// IDs are UUID strings. Times are UTC.
package domain

import (
	"encoding/json"
	"time"
)

// DemoUserID is the single seeded user. There is no auth today.
const DemoUserID = "00000000-0000-0000-0000-000000000001"

type ClaimLabel string

const (
	LabelSupported    ClaimLabel = "supported"
	LabelPartial      ClaimLabel = "partial"
	LabelContradicted ClaimLabel = "contradicted"
	LabelAbsent       ClaimLabel = "absent"
)

type Grade string

const (
	GradeAgain Grade = "again"
	GradeHard  Grade = "hard"
	GradeGood  Grade = "good"
	GradeEasy  Grade = "easy"
)

type EventType string

const (
	EventCardReview     EventType = "card_review"
	EventTeachBack      EventType = "teach_back"
	EventManualOverride EventType = "manual_override"
)

type SourceKind string

const (
	SourceCourse    SourceKind = "course"
	SourceAllowlist SourceKind = "allowlist"
)

type Level string

const (
	LevelBeginner     Level = "beginner"
	LevelIntermediate Level = "intermediate"
	LevelAdvanced     Level = "advanced"
)

type PlanKind string

const (
	PlanReview PlanKind = "review"
	PlanTeach  PlanKind = "teach"
	PlanNew    PlanKind = "new"
)

type Source struct {
	ID         string     `json:"id"`
	Kind       SourceKind `json:"kind"`
	Title      string     `json:"title"`
	URL        string     `json:"url,omitempty"`
	Publisher  string     `json:"publisher,omitempty"`
	License    string     `json:"license,omitempty"`
	ReviewedBy string     `json:"reviewed_by,omitempty"`
	ReviewedAt *time.Time `json:"reviewed_at,omitempty"`
	Text       string     `json:"text"`
}

type Subject struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	WeeklyHours float64    `json:"weekly_hours"`
	Level       Level      `json:"level"`
	ExamDate    *time.Time `json:"exam_date,omitempty"`
	Weight      float64    `json:"weight"`
}

type Concept struct {
	ID        string  `json:"id"`
	SubjectID string  `json:"subject_id"`
	Name      string  `json:"name"`
	Claims    []Claim `json:"claims"`
}

type Claim struct {
	ID        string `json:"id"`
	ConceptID string `json:"concept_id"`
	Text      string `json:"text"`
	SourceID  string `json:"source_id"`
	Locator   string `json:"locator"`
	Required  bool   `json:"required"`
}

// TaughtClaim is what the human said, in their own words. It is the only
// knowledge the AI student ever receives.
type TaughtClaim struct {
	ID          string `json:"id"`
	SessionID   string `json:"session_id"`
	ConceptID   string `json:"concept_id"`
	StudentText string `json:"student_text"`
	TurnNo      int    `json:"turn_no"`
}

// Citation quotes an approved source. Code verifies Quote is an exact
// substring of that source's text before any correction is shown.
type Citation struct {
	SourceID string `json:"source_id"`
	Quote    string `json:"quote"`
	Locator  string `json:"locator"`
}

type ClaimResult struct {
	ClaimID     *string    `json:"claim_id"`
	StudentText string     `json:"student_text"`
	Label       ClaimLabel `json:"label"`
	Citation    *Citation  `json:"citation"`
	Correction  *string    `json:"correction"`
	// ProbeHint steers the next question. It never contains source text.
	ProbeHint *string `json:"probe_hint"`
}

type ReviewEvent struct {
	ID              string          `json:"id"`
	EventType       EventType       `json:"event_type"`
	ConceptID       string          `json:"concept_id"`
	CardID          *string         `json:"card_id"`
	Grade           *Grade          `json:"grade"`
	DueBefore       time.Time       `json:"due_before"`
	DueAfter        time.Time       `json:"due_after"`
	StabilityBefore float64         `json:"stability_before"`
	StabilityAfter  float64         `json:"stability_after"`
	Payload         json.RawMessage `json:"payload"`
	CreatedAt       time.Time       `json:"created_at"`
}

type PlanBlock struct {
	ID        string    `json:"id"`
	SubjectID string    `json:"subject_id"`
	ConceptID *string   `json:"concept_id"`
	Kind      PlanKind  `json:"kind"`
	StartsAt  time.Time `json:"starts_at"`
	Minutes   int       `json:"minutes"`
	Reason    string    `json:"reason"`
	Locked    bool      `json:"locked"`
}

// Card is not in the U0 prompt's type list but the cards table and the
// PATCH /cards/{id} endpoint need it. Cards carry no schedule of their own.
// The concept is the FSRS unit.
type Card struct {
	ID        string  `json:"id"`
	ConceptID string  `json:"concept_id"`
	ClaimID   *string `json:"claim_id"`
	Front     string  `json:"front"`
	Back      string  `json:"back"`
}

// Supporting shapes for the interfaces.

type Message struct {
	Role    string `json:"role"` // "user" or "assistant"
	Content string `json:"content"`
}

type LLMRequest struct {
	System   string    `json:"system"`
	Messages []Message `json:"messages"`
	Model    string    `json:"model,omitempty"`
}

// Delta is one streamed chunk. Err is set on a terminal failure.
type Delta struct {
	Text string
	Err  error
}

type TeachSession struct {
	ID        string `json:"id"`
	ConceptID string `json:"concept_id"`
}

type Report struct {
	SessionID string        `json:"session_id"`
	ConceptID string        `json:"concept_id"`
	Results   []ClaimResult `json:"results"`
	// Schedule is the teach_back event written after the attempt, so the
	// date change is visible. Nil until the scheduler has run.
	Schedule *ReviewEvent `json:"schedule"`
}

type CheckInput struct {
	Concept    Concept       `json:"concept"`
	Sources    []Source      `json:"sources"`
	Taught     []TaughtClaim `json:"taught"`
	Transcript string        `json:"transcript"`
	LaTeX      string        `json:"latex"`
}

type DueItem struct {
	ConceptID   string    `json:"concept_id"`
	ConceptName string    `json:"concept_name"`
	CardID      *string   `json:"card_id"`
	Card        *Card     `json:"card"` // set when CardID is, so the review screen can show it
	DueAt       time.Time `json:"due_at"`
}

type Plan struct {
	WeekStart time.Time   `json:"week_start"`
	Blocks    []PlanBlock `json:"blocks"`
}

// ConceptProgress keeps recall (cards) and explanation (teach-back) apart.
// A nil value means no data yet.
type ConceptProgress struct {
	ConceptID   string     `json:"concept_id"`
	Name        string     `json:"name"`
	Recall      *float64   `json:"recall"`
	Explanation *float64   `json:"explanation"`
	NextDue     *time.Time `json:"next_due"`
}

type SubjectProgress struct {
	SubjectID     string            `json:"subject_id"`
	Name          string            `json:"name"`
	Concepts      []ConceptProgress `json:"concepts"`
	NextWeekHours float64           `json:"next_week_hours"`
}

type FurtherPlanning struct {
	WeekStart time.Time         `json:"week_start"`
	Subjects  []SubjectProgress `json:"subjects"`
}

type DigestItem struct {
	SubjectID   string `json:"subject_id"`
	SubjectName string `json:"subject_name"`
	DueCount    int    `json:"due_count"`
	Minutes     int    `json:"minutes"`
}

type Digest struct {
	Date  time.Time    `json:"date"`
	Items []DigestItem `json:"items"`
}
