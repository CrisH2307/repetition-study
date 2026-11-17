package fakes

import (
	"context"
	"encoding/hex"
	"sync"
	"time"

	"github.com/CrisH2307/repetition-study/internal/domain"
)

// Contract v5 fakes. In memory, scoped to the user id they are given.

var (
	_ domain.ShareStore = (*ShareStore)(nil)
	_ domain.Sharing    = (*Sharing)(nil)
)

// ShareStore keeps links keyed by id and by the hex of the token hash.
type ShareStore struct {
	mu          sync.Mutex
	ids         ids
	shares      map[string]*domain.Share
	byHash      map[string]string
	Instructors map[string]bool
}

func (s *ShareStore) CreateShare(_ context.Context, user, subjectID string, h []byte, src bool) (domain.Share, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.shares == nil {
		s.shares, s.byHash = map[string]*domain.Share{}, map[string]string{}
	}
	sh := &domain.Share{ID: s.ids.next("share"), SubjectID: subjectID, IncludesSources: src, CreatedAt: Epoch, OwnerID: user}
	s.shares[sh.ID], s.byHash[hex.EncodeToString(h)] = sh, sh.ID
	return *sh, nil
}

func (s *ShareStore) RevokeShare(_ context.Context, user, id string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	sh, ok := s.shares[id]
	if !ok || sh.OwnerID != user {
		return domain.ErrNotFound
	}
	if sh.RevokedAt == nil {
		sh.RevokedAt = &at
	}
	return nil
}

func (s *ShareStore) ShareByToken(_ context.Context, h []byte) (domain.Share, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sh, ok := s.shares[s.byHash[hex.EncodeToString(h)]]
	if !ok || sh.RevokedAt != nil {
		return domain.Share{}, domain.ErrNotFound
	}
	return *sh, nil
}

func (s *ShareStore) SetInstructor(_ context.Context, user string, on bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Instructors == nil {
		s.Instructors = map[string]bool{}
	}
	s.Instructors[user] = on
	return nil
}

func (s *ShareStore) instructor(user string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Instructors[user]
}

// Sharing previews what tests put in Packs (by subject id). Import adds every
// pack concept and lists every claim as needing a quote; it creates nothing
// (the real one uses CourseStore and CourseNotes).
type Sharing struct {
	Store *ShareStore
	Packs map[string]domain.PackPreview
	Clock domain.Clock

	mu  sync.Mutex
	ids ids
}

func (s *Sharing) Create(ctx context.Context, user, subjectID string, src bool) (domain.NewShare, error) {
	if src && !s.Store.instructor(user) {
		return domain.NewShare{}, &domain.UserError{Msg: "Only instructors can share their notes."}
	}
	s.mu.Lock()
	tok := s.ids.next("token")
	s.mu.Unlock()
	sh, err := s.Store.CreateShare(ctx, user, subjectID, []byte(tok), src)
	return domain.NewShare{Share: sh, Token: tok}, err
}

func (s *Sharing) Revoke(ctx context.Context, user, id string) error {
	at := Epoch
	if s.Clock != nil {
		at = s.Clock.Now()
	}
	return s.Store.RevokeShare(ctx, user, id, at)
}

func (s *Sharing) Preview(ctx context.Context, token string) (domain.PackPreview, error) {
	sh, err := s.Store.ShareByToken(ctx, []byte(token))
	if err != nil {
		return domain.PackPreview{}, err
	}
	p, ok := s.Packs[sh.SubjectID]
	if !ok {
		return domain.PackPreview{}, domain.ErrNotFound
	}
	p.Instructor = sh.IncludesSources && s.Store.instructor(sh.OwnerID)
	if !sh.IncludesSources || p.Sources == nil {
		p.Sources = []string{}
	}
	return p, nil
}

func (s *Sharing) Import(ctx context.Context, user, token, subjectID string) (domain.PackImport, error) {
	p, err := s.Preview(ctx, token)
	if err != nil {
		return domain.PackImport{}, err
	}
	out := domain.PackImport{SubjectID: subjectID, NeedsQuote: []domain.NeedsQuote{}}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, c := range p.Concepts {
		id := s.ids.next("concept")
		out.ConceptsAdded++
		for _, cl := range c.Claims {
			out.NeedsQuote = append(out.NeedsQuote, domain.NeedsQuote{ConceptID: id, Claim: cl})
		}
	}
	return out, nil
}
