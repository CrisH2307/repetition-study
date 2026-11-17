package domain

import (
	"context"
	"errors"
	"time"
)

// Contract v5 (Phase 3, D49 to D51). Billing in test mode and claim-only
// pack sharing with instructor packs. Both stay off until Cris turns them on.

// Billing (D49).

type PlanName string

const (
	PlanMonth PlanName = "month"
	PlanTerm  PlanName = "term" // one-time payment, no subscription
)

type EntitlementStatus string

const (
	EntitlementActive   EntitlementStatus = "active"
	EntitlementPastDue  EntitlementStatus = "past_due"
	EntitlementCanceled EntitlementStatus = "canceled"
)

// Entitlement is the one row per user that says what they paid for. Provider
// ids are internal and never sent to the browser.
type Entitlement struct {
	Plan           PlanName          `json:"plan"`
	Status         EntitlementStatus `json:"status"`
	PeriodEnd      time.Time         `json:"period_end"`
	CustomerID     string            `json:"-"`
	SubscriptionID *string           `json:"-"` // set for month, nil for term
}

// BillingStatus is GET /v1/billing. Enabled is false while BILLING_ENABLED is
// off; then Entitled is always true and Plan is nil.
type BillingStatus struct {
	Enabled   bool               `json:"enabled"`
	Entitled  bool               `json:"entitled"`
	Plan      *PlanName          `json:"plan"`
	Status    *EntitlementStatus `json:"status"`
	PeriodEnd *time.Time         `json:"period_end"`
}

// BillingEvent is one verified webhook event as received.
type BillingEvent struct {
	EventID    string
	Type       string
	ReceivedAt time.Time
	Payload    []byte // raw JSON object
}

// ErrBadSignature means the webhook signature header is missing, wrong or too
// old. The handler answers 400 and changes nothing.
var ErrBadSignature = errors.New("bad webhook signature")

// BillingStore persists entitlements and webhook events. P3-BILLING implements it.
type BillingStore interface {
	// Entitlement returns ErrNotFound when the user never paid.
	Entitlement(ctx context.Context, userID string) (Entitlement, error)
	// UserByCustomer maps a provider customer id to the user, ErrNotFound if none.
	UserByCustomer(ctx context.Context, customerID string) (string, error)
	// ApplyEvent records ev and, when e is not nil, upserts the user's
	// entitlement, in one transaction. applied is false, and nothing changes,
	// when ev.EventID was seen before.
	ApplyEvent(ctx context.Context, ev BillingEvent, userID string, e *Entitlement) (applied bool, err error)
}

// Billing is Stripe Checkout and webhooks over net/http (D49). P3-BILLING implements it.
type Billing interface {
	Status(ctx context.Context, userID string) (BillingStatus, error)
	// CheckoutURL returns the hosted Checkout page for the plan. A UserError
	// while billing is off.
	CheckoutURL(ctx context.Context, userID string, plan PlanName) (string, error)
	// HandleWebhook verifies signature over the raw body, then applies the
	// event once. ErrBadSignature when verification fails.
	HandleWebhook(ctx context.Context, body []byte, signature string) error
	// Entitled is always true while billing is off.
	Entitled(ctx context.Context, userID string) (bool, error)
}

// Sharing (D50, D51).

// Share is one share link. The token is only returned once, at creation.
type Share struct {
	ID              string     `json:"id"`
	SubjectID       string     `json:"subject_id"`
	IncludesSources bool       `json:"includes_sources"`
	CreatedAt       time.Time  `json:"created_at"`
	RevokedAt       *time.Time `json:"revoked_at"`
	OwnerID         string     `json:"-"`
}

// NewShare is POST /v1/courses/{id}/share: the share and its one-time token.
type NewShare struct {
	Share
	Token string `json:"token"`
}

// PackClaim is a claim as it travels in a pack: text and kind only, never a
// quote or course text (D50). A claim stored without a kind travels as
// statement.
type PackClaim struct {
	Text     string    `json:"text"`
	Kind     ClaimKind `json:"kind"`
	Required bool      `json:"required"`
}

type PackConcept struct {
	Name   string      `json:"name"`
	Claims []PackClaim `json:"claims"`
}

// PackPreview is GET /v1/shared/{token}. Instructor is true for a pack whose
// owner is an instructor and that includes their notes (the "instructor
// approved" badge). Sources lists those notes' file names only.
type PackPreview struct {
	CourseName string        `json:"course_name"`
	OwnerName  string        `json:"owner_name"`
	Instructor bool          `json:"instructor"`
	Concepts   []PackConcept `json:"concepts"`
	Sources    []string      `json:"sources"`
}

// NeedsQuote is a pack claim that was not added because no quote for it was
// found on the receiver's Ready notes. The user adds it by hand.
type NeedsQuote struct {
	ConceptID string    `json:"concept_id"`
	Claim     PackClaim `json:"claim"`
}

// PackImport is POST /v1/shared/{token}/import.
type PackImport struct {
	SubjectID     string       `json:"subject_id"`
	ConceptsAdded int          `json:"concepts_added"`
	ClaimsAdded   int          `json:"claims_added"`
	UploadsCopied int          `json:"uploads_copied"` // instructor packs only
	NeedsQuote    []NeedsQuote `json:"needs_quote"`
}

// ShareStore persists share links and the instructor flag. P3-SHARE implements it.
type ShareStore interface {
	// CreateShare stores the hash of a new token for the user's course.
	CreateShare(ctx context.Context, userID, subjectID string, tokenHash []byte, includesSources bool) (Share, error)
	// RevokeShare sets revoked_at once; ErrNotFound for another user's share.
	RevokeShare(ctx context.Context, userID, shareID string, at time.Time) error
	// ShareByToken returns ErrNotFound for an unknown or revoked token.
	ShareByToken(ctx context.Context, tokenHash []byte) (Share, error)
	// SetInstructor is for cmd/instructor only (D51), never the API.
	SetInstructor(ctx context.Context, userID string, on bool) error
}

// Sharing is the pack logic. P3-SHARE implements it.
type Sharing interface {
	// Create makes a link for the user's course. includesSources needs
	// User.IsInstructor, else a UserError.
	Create(ctx context.Context, userID, subjectID string, includesSources bool) (NewShare, error)
	Revoke(ctx context.Context, userID, shareID string) error
	// Preview needs no sign-in. ErrNotFound for an unknown or revoked token.
	Preview(ctx context.Context, token string) (PackPreview, error)
	// Import adds the pack to the user's course. Each claim goes through
	// CourseNotes.AddClaim with a quote from the receiver's own Ready notes, or
	// from the copied instructor notes; claims with no quote come back in NeedsQuote.
	Import(ctx context.Context, userID, token, subjectID string) (PackImport, error)
}
