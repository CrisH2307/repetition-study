// Package course is the course logic: reading uploads into pages, the page
// check (D27) and the claim checklist (D28). Quotes are verified only by
// internal/cite. Course text is untrusted data and no model sees it here.
package course

import (
	"context"
	"strings"
	"time"

	"github.com/CrisH2307/repetition-study/internal/cite"
	"github.com/CrisH2307/repetition-study/internal/domain"
)

// Notes implements domain.CourseNotes over a CourseStore.
type Notes struct {
	Store domain.CourseStore
	Now   func() time.Time // nil means time.Now
}

var _ domain.CourseNotes = Notes{}

func (n Notes) now() time.Time {
	if n.Now != nil {
		return n.Now()
	}
	return time.Now()
}

func (n Notes) ReadUpload(ctx context.Context, userID, uploadID string) error {
	pdf, pasted, err := n.Store.UploadFile(ctx, userID, uploadID)
	if err != nil {
		return err
	}
	fail := func(msg string) error {
		return n.Store.SetUploadStatus(ctx, userID, uploadID, domain.UploadNeedsChecking, msg)
	}
	var texts []string
	var images [][]byte
	if len(pdf) > 0 {
		var rerr error
		if texts, images, rerr = readPDF(ctx, pdf); rerr != nil {
			return fail(rerr.Error())
		}
	} else {
		texts = []string{pasted} // pasted text is one page
	}
	pages := make([]domain.Page, len(texts))
	for i, t := range texts {
		pages[i] = domain.Page{Number: i + 1, RawText: t, EditedText: t}
	}
	if err := n.Store.SavePages(ctx, userID, uploadID, pages, images); err != nil {
		return err
	}
	return n.Store.SetUploadStatus(ctx, userID, uploadID, domain.UploadNeedsChecking, "")
}

func (n Notes) EditPage(ctx context.Context, userID, pageID, text string) (domain.Page, error) {
	return n.Store.EditPage(ctx, userID, pageID, text)
}

// MarkReady turns the saved text of the page into one source block (the page
// is the unit the student checked), so a quote may span lines but not pages.
func (n Notes) MarkReady(ctx context.Context, userID, pageID string) (domain.Page, error) {
	p, err := n.Store.GetPage(ctx, userID, pageID)
	if err != nil {
		return domain.Page{}, err
	}
	norm := cite.Normalize(p.EditedText)
	if norm == "" {
		return domain.Page{}, &domain.UserError{Msg: "This page has no text. Type the text in before marking it Ready."}
	}
	at := n.now()
	b := domain.SourceBlock{Page: &p.Number, Kind: domain.BlockText, TextMD: p.EditedText, TextNorm: norm, VerifiedBy: userID, VerifiedAt: &at}
	return n.Store.SetPageReady(ctx, userID, pageID, []domain.SourceBlock{b})
}

// MarkNotReady removes the page's blocks. The store refuses while a live claim
// anchors to one; this layer turns any other failure into itself unchanged.
func (n Notes) MarkNotReady(ctx context.Context, userID, pageID string) (domain.Page, error) {
	return n.Store.SetPageNotReady(ctx, userID, pageID)
}

// verify checks the claim input against Ready blocks, the only quotable text.
func verify(blocks []domain.SourceBlock, in domain.ClaimInput) error {
	if strings.TrimSpace(in.Text) == "" {
		return &domain.UserError{Msg: "Write the claim in your own words first."}
	}
	switch in.Kind {
	case domain.KindDefinition, domain.KindStatement, domain.KindCondition, domain.KindMethod:
	default:
		return &domain.UserError{Msg: "Pick what kind of claim this is: definition, statement, condition or method."}
	}
	for _, b := range blocks {
		if b.ID == in.Anchor.BlockID {
			if cite.Verify(b.TextMD, in.Anchor.Quote) {
				return nil
			}
			return &domain.UserError{Msg: "That quote is not word for word on the page, or it is too short. Copy it from the page text."}
		}
	}
	return &domain.UserError{Msg: "That quote is not on a Ready page of this course. Mark the page Ready first."}
}

func (n Notes) AddClaim(ctx context.Context, userID, conceptID string, in domain.ClaimInput) (domain.Claim, error) {
	c, err := n.Store.GetConcept(ctx, userID, conceptID)
	if err != nil {
		return domain.Claim{}, err
	}
	blocks, err := n.Store.ReadyBlocks(ctx, userID, c.SubjectID)
	if err != nil {
		return domain.Claim{}, err
	}
	if err := verify(blocks, in); err != nil {
		return domain.Claim{}, err
	}
	return n.Store.CreateClaim(ctx, userID, conceptID, in)
}

// EditClaim checks the new anchor against the user's Ready blocks in all their
// courses: CourseStore has no way to find a claim's course.
// ponytail: a claim could be re-anchored to the user's other course. Add
// CourseStore.GetClaim (contract change, ask Cris) to pin it to its own course.
func (n Notes) EditClaim(ctx context.Context, userID, claimID string, in domain.ClaimInput) (domain.Claim, error) {
	courses, err := n.Store.ListCourses(ctx, userID)
	if err != nil {
		return domain.Claim{}, err
	}
	var blocks []domain.SourceBlock
	for _, c := range courses {
		bs, err := n.Store.ReadyBlocks(ctx, userID, c.ID)
		if err != nil {
			return domain.Claim{}, err
		}
		blocks = append(blocks, bs...)
	}
	if err := verify(blocks, in); err != nil {
		return domain.Claim{}, err
	}
	return n.Store.UpdateClaim(ctx, userID, claimID, in)
}

func (n Notes) DeleteClaim(ctx context.Context, userID, claimID string) error {
	return n.Store.DeleteClaim(ctx, userID, claimID)
}
