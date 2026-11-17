// Package auth is sign-in by invite link (D29). Tokens are 32 random bytes;
// only their SHA-256 reaches the store.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"time"

	"github.com/CrisH2307/repetition-study/internal/domain"
)

// SessionTTL is how long a sign-in lasts.
const SessionTTL = 30 * 24 * time.Hour

var _ domain.Auth = (*Service)(nil)

// Service implements domain.Auth over a domain.AuthStore.
type Service struct {
	Store domain.AuthStore
	Clock domain.Clock // nil means time.Now
}

func (s *Service) now() time.Time {
	if s.Clock == nil {
		return time.Now()
	}
	return s.Clock.Now()
}

func newToken() string {
	b := make([]byte, 32)
	rand.Read(b) // never returns an error (crypto/rand docs, Go 1.24+)
	return base64.RawURLEncoding.EncodeToString(b)
}

func hash(token string) []byte {
	h := sha256.Sum256([]byte(token))
	return h[:]
}

func (s *Service) NewInvite(ctx context.Context, name string) (string, error) {
	if name == "" {
		return "", &domain.UserError{Msg: "An invite needs a name."}
	}
	t := newToken()
	if _, err := s.Store.CreateInvite(ctx, name, hash(t)); err != nil {
		return "", err
	}
	return t, nil
}

// Redeem signs in the invite's user, creating it on first use. An invite
// always maps to the same user, so a reused link never makes a second account.
func (s *Service) Redeem(ctx context.Context, invite string) (domain.User, string, time.Time, error) {
	if invite == "" {
		return domain.User{}, "", time.Time{}, domain.ErrNotFound
	}
	now := s.now()
	u, err := s.Store.RedeemInvite(ctx, hash(invite), now)
	if err != nil {
		return domain.User{}, "", time.Time{}, err
	}
	t, exp := newToken(), now.Add(SessionTTL)
	if err := s.Store.CreateSession(ctx, u.ID, hash(t), exp); err != nil {
		return domain.User{}, "", time.Time{}, err
	}
	return u, t, exp, nil
}

func (s *Service) UserFor(ctx context.Context, session string) (domain.User, error) {
	if session == "" {
		return domain.User{}, domain.ErrNotFound
	}
	return s.Store.SessionUser(ctx, hash(session), s.now())
}

func (s *Service) SignOut(ctx context.Context, session string) error {
	if session == "" {
		return nil
	}
	return s.Store.DeleteSession(ctx, hash(session))
}
