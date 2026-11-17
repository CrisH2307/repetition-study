package store

import (
	"context"
	"time"

	"github.com/CrisH2307/repetition-study/internal/domain"
)

var _ domain.ShareStore = (*Store)(nil)

func (s *Store) CreateShare(ctx context.Context, userID, subjectID string, tokenHash []byte, includesSources bool) (domain.Share, error) {
	if _, err := getCourse(ctx, s.pool, userID, subjectID); err != nil {
		return domain.Share{}, err
	}
	sh := domain.Share{SubjectID: subjectID, IncludesSources: includesSources, OwnerID: userID}
	err := s.pool.QueryRow(ctx, `INSERT INTO shared_packs (owner_id, subject_id, token_hash, includes_sources)
		VALUES ($1, $2, $3, $4) RETURNING id::text, created_at`, userID, subjectID, tokenHash, includesSources).
		Scan(&sh.ID, &sh.CreatedAt)
	return sh, err
}

// RevokeShare keeps the first revoked_at. Another user's id gives ErrNotFound.
func (s *Store) RevokeShare(ctx context.Context, userID, shareID string, at time.Time) error {
	return affected(s.pool.Exec(ctx, `UPDATE shared_packs SET revoked_at = COALESCE(revoked_at, GREATEST($3, created_at))
		WHERE id = $1 AND owner_id = $2`, shareID, userID, at))
}

// ShareByToken also hides links whose course was archived.
func (s *Store) ShareByToken(ctx context.Context, tokenHash []byte) (domain.Share, error) {
	var sh domain.Share
	err := s.pool.QueryRow(ctx, `SELECT p.id::text, p.subject_id::text, p.includes_sources, p.created_at, p.owner_id::text
		FROM shared_packs p JOIN subjects s ON s.id = p.subject_id
		WHERE p.token_hash = $1 AND p.revoked_at IS NULL AND s.archived_at IS NULL`, tokenHash).
		Scan(&sh.ID, &sh.SubjectID, &sh.IncludesSources, &sh.CreatedAt, &sh.OwnerID)
	return sh, nf(err)
}

func (s *Store) SetInstructor(ctx context.Context, userID string, on bool) error {
	return affected(s.pool.Exec(ctx, `UPDATE users SET is_instructor = $2 WHERE id = $1`, userID, on))
}

// UserByID is for the share logic: the owner's name and instructor flag.
func (s *Store) UserByID(ctx context.Context, id string) (domain.User, error) {
	var u domain.User
	err := s.pool.QueryRow(ctx, `SELECT id::text, name, time_zone, is_instructor FROM users WHERE id = $1`, id).
		Scan(&u.ID, &u.Name, &u.TimeZone, &u.IsInstructor)
	return u, nf(err)
}
