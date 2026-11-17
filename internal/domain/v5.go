package domain

import (
	"context"
	"time"
)

// Contract v5 (Phase 3, D50, D51). Claim-only pack sharing with instructor
// packs. Billing (D49) is deferred by Cris.

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
