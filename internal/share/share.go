// Package share is claim-only pack sharing (D50) with instructor packs (D51).
// A pack carries concept names, claim text and kind. Quotes and course text
// never travel; the receiver's claims are anchored by the existing course
// logic (D28), so the quote check is the same one as for a typed claim.
package share

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"
	"time"
	"unicode"

	"github.com/CrisH2307/repetition-study/internal/cite"
	"github.com/CrisH2307/repetition-study/internal/domain"
)

// Users finds a user by id. *store.Store has it.
type Users interface {
	UserByID(ctx context.Context, id string) (domain.User, error)
}

type Service struct {
	Store   domain.ShareStore
	Users   Users
	Courses domain.CourseStore
	Review  domain.ReviewStore
	Notes   domain.CourseNotes
	Now     func() time.Time // nil means time.Now
}

var _ domain.Sharing = (*Service)(nil)

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now().UTC()
}

func hash(token string) []byte { h := sha256.Sum256([]byte(token)); return h[:] }

func (s *Service) Create(ctx context.Context, userID, subjectID string, includesSources bool) (domain.NewShare, error) {
	if _, err := s.Courses.GetCourse(ctx, userID, subjectID); err != nil {
		return domain.NewShare{}, err
	}
	if includesSources {
		u, err := s.Users.UserByID(ctx, userID)
		if err != nil {
			return domain.NewShare{}, err
		}
		if !u.IsInstructor {
			return domain.NewShare{}, &domain.UserError{Msg: "Only instructors can share their notes."}
		}
	}
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return domain.NewShare{}, err
	}
	tok := base64.RawURLEncoding.EncodeToString(b)
	sh, err := s.Store.CreateShare(ctx, userID, subjectID, hash(tok), includesSources)
	if err != nil {
		return domain.NewShare{}, err
	}
	return domain.NewShare{Share: sh, Token: tok}, nil
}

func (s *Service) Revoke(ctx context.Context, userID, shareID string) error {
	return s.Store.RevokeShare(ctx, userID, shareID, s.now())
}

func (s *Service) Preview(ctx context.Context, token string) (domain.PackPreview, error) {
	p, _, err := s.preview(ctx, token)
	return p, err
}

func (s *Service) preview(ctx context.Context, token string) (domain.PackPreview, domain.Share, error) {
	sh, err := s.Store.ShareByToken(ctx, hash(token))
	if err != nil {
		return domain.PackPreview{}, sh, err
	}
	owner, err := s.Users.UserByID(ctx, sh.OwnerID)
	if err != nil {
		return domain.PackPreview{}, sh, err
	}
	course, err := s.Courses.GetCourse(ctx, sh.OwnerID, sh.SubjectID)
	if err != nil {
		return domain.PackPreview{}, sh, err
	}
	view, err := s.Review.CourseView(ctx, sh.OwnerID, sh.SubjectID)
	if err != nil {
		return domain.PackPreview{}, sh, err
	}
	p := domain.PackPreview{
		CourseName: course.Name,
		OwnerName:  owner.Name,
		Instructor: sh.IncludesSources && owner.IsInstructor,
		Concepts:   []domain.PackConcept{},
		Sources:    []string{},
	}
	for _, card := range view.Cards {
		c, err := s.Courses.GetConcept(ctx, sh.OwnerID, card.ConceptID)
		if err != nil {
			return domain.PackPreview{}, sh, err
		}
		pc := domain.PackConcept{Name: c.Name, Claims: []domain.PackClaim{}}
		for _, cl := range c.Claims {
			k := domain.KindStatement
			if cl.Kind != nil {
				k = *cl.Kind
			}
			pc.Claims = append(pc.Claims, domain.PackClaim{Text: cl.Text, Kind: k, Required: cl.Required})
		}
		p.Concepts = append(p.Concepts, pc)
	}
	if p.Instructor {
		ups, err := s.Courses.ListUploads(ctx, sh.OwnerID, sh.SubjectID)
		if err != nil {
			return domain.PackPreview{}, sh, err
		}
		for _, u := range ups {
			if u.PagesReady > 0 {
				p.Sources = append(p.Sources, u.Filename)
			}
		}
	}
	return p, sh, nil
}

func (s *Service) Import(ctx context.Context, userID, token, subjectID string) (domain.PackImport, error) {
	if _, err := s.Courses.GetCourse(ctx, userID, subjectID); err != nil {
		return domain.PackImport{}, err
	}
	p, sh, err := s.preview(ctx, token)
	if err != nil {
		return domain.PackImport{}, err
	}
	out := domain.PackImport{SubjectID: subjectID, NeedsQuote: []domain.NeedsQuote{}}
	if p.Instructor {
		if out.UploadsCopied, err = s.copyNotes(ctx, sh.OwnerID, sh.SubjectID, userID, subjectID); err != nil {
			return out, err
		}
	}
	blocks, err := s.Courses.ReadyBlocks(ctx, userID, subjectID)
	if err != nil {
		return out, err
	}
	have := map[string]string{} // lower-case concept name -> id
	if v, err := s.Review.CourseView(ctx, userID, subjectID); err == nil {
		for _, c := range v.Cards {
			have[strings.ToLower(c.Name)] = c.ConceptID
		}
	}
	for _, pc := range p.Concepts {
		id, ok := have[strings.ToLower(pc.Name)]
		if !ok {
			c, err := s.Courses.CreateConcept(ctx, userID, subjectID, pc.Name)
			if err != nil {
				return out, err
			}
			id = c.ID
			have[strings.ToLower(pc.Name)] = id
			out.ConceptsAdded++
		}
		for _, cl := range pc.Claims {
			ok, err := s.addClaim(ctx, userID, id, cl, blocks)
			if err != nil {
				return out, err
			}
			if ok {
				out.ClaimsAdded++
				continue
			}
			out.NeedsQuote = append(out.NeedsQuote, domain.NeedsQuote{ConceptID: id, Claim: cl})
		}
	}
	return out, nil
}

// addClaim anchors cl to the best matching sentence of the receiver's Ready
// notes through CourseNotes.AddClaim. False means no quote was found or the
// course logic refused it with a UserError; nothing is saved then.
func (s *Service) addClaim(ctx context.Context, userID, conceptID string, cl domain.PackClaim, blocks []domain.SourceBlock) (bool, error) {
	blockID, quote := findQuote(cl.Text, blocks)
	if quote == "" {
		return false, nil
	}
	_, err := s.Notes.AddClaim(ctx, userID, conceptID, domain.ClaimInput{
		Text: cl.Text, Kind: cl.Kind, Required: cl.Required,
		Anchor: domain.Anchor{BlockID: blockID, Quote: quote},
	})
	var ue *domain.UserError
	if errors.As(err, &ue) {
		return false, nil
	}
	return err == nil, err
}

// copyNotes copies the owner's Ready pages into the receiver's course as new
// Ready notes (instructor packs only, D51). One upload per owner upload, its
// Ready pages joined, so the receiver's own page check applies to the copy.
// ponytail: page breaks are lost in the copy; locators say p.1.
func (s *Service) copyNotes(ctx context.Context, owner, from, user, to string) (int, error) {
	ups, err := s.Courses.ListUploads(ctx, owner, from)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, u := range ups {
		pages, err := s.Courses.ListPages(ctx, owner, u.ID)
		if err != nil {
			return n, err
		}
		var texts []string
		for _, pg := range pages {
			if pg.Status == domain.PageReady && strings.TrimSpace(pg.EditedText) != "" {
				texts = append(texts, pg.EditedText)
			}
		}
		if len(texts) == 0 {
			continue
		}
		nu, err := s.Courses.CreateUpload(ctx, user, domain.Upload{SubjectID: to, Filename: u.Filename}, nil, strings.Join(texts, "\n\n"))
		if err != nil {
			return n, err
		}
		if err := s.Notes.ReadUpload(ctx, user, nu.ID); err != nil {
			return n, err
		}
		np, err := s.Courses.ListPages(ctx, user, nu.ID)
		if err != nil {
			return n, err
		}
		for _, pg := range np {
			if _, err := s.Notes.MarkReady(ctx, user, pg.ID); err != nil {
				return n, err
			}
		}
		n++
	}
	return n, nil
}

// findQuote picks the sentence of the notes that shares the most words with
// the claim, if at least half the claim's longer words are in it. The quote is
// cut from the block text and re-checked with cite.Verify; AddClaim checks it
// again (D28). No model is involved.
func findQuote(claim string, blocks []domain.SourceBlock) (blockID, quote string) {
	want := words(claim)
	if len(want) == 0 {
		return "", ""
	}
	best := 0.0
	for _, b := range blocks {
		for _, sent := range sentences(b.TextMD) {
			got := words(sent)
			hit := 0
			for w := range want {
				if got[w] {
					hit++
				}
			}
			if sc := float64(hit) / float64(len(want)); sc >= 0.5 && sc > best && cite.Verify(b.TextMD, sent) {
				best, blockID, quote = sc, b.ID, sent
			}
		}
	}
	return
}

func words(s string) map[string]bool {
	m := map[string]bool{}
	for _, w := range strings.FieldsFunc(strings.ToLower(s), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) }) {
		if len(w) >= 4 {
			m[w] = true
		}
	}
	return m
}

func sentences(s string) []string {
	var out []string
	start := 0
	for i, r := range s {
		if r == '.' || r == '!' || r == '?' || r == '\n' {
			if t := strings.TrimSpace(s[start : i+1]); t != "" {
				out = append(out, t)
			}
			start = i + 1
		}
	}
	if t := strings.TrimSpace(s[start:]); t != "" {
		out = append(out, t)
	}
	return out
}
