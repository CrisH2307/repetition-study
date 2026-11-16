package store

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/CrisH2307/repetition-study/internal/domain"
)

func (s *Store) CreateInvite(ctx context.Context, name string, tokenHash []byte) (domain.Invite, error) {
	inv := domain.Invite{Name: name}
	err := s.pool.QueryRow(ctx, `INSERT INTO invites (token_hash, name) VALUES ($1, $2) RETURNING id::text, created_at`,
		tokenHash, name).Scan(&inv.ID, &inv.CreatedAt)
	return inv, err
}

// RedeemInvite creates the user on first use. The invite row is locked, so two
// clicks at once make one user.
func (s *Store) RedeemInvite(ctx context.Context, tokenHash []byte, now time.Time) (domain.User, error) {
	var u domain.User
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		var name string
		var userID *string
		if err := tx.QueryRow(ctx, `SELECT name, user_id::text FROM invites WHERE token_hash = $1 FOR UPDATE`,
			tokenHash).Scan(&name, &userID); err != nil {
			return nf(err)
		}
		if userID == nil {
			var id string
			if err := tx.QueryRow(ctx, `INSERT INTO users (name) VALUES ($1) RETURNING id::text`, name).Scan(&id); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `UPDATE invites SET used_at = $2, user_id = $3 WHERE token_hash = $1`,
				tokenHash, now, id); err != nil {
				return err
			}
			userID = &id
		}
		return tx.QueryRow(ctx, `SELECT id::text, name, time_zone FROM users WHERE id = $1`, *userID).
			Scan(&u.ID, &u.Name, &u.TimeZone)
	})
	return u, err
}

func (s *Store) CreateSession(ctx context.Context, userID string, tokenHash []byte, expiresAt time.Time) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO auth_sessions (token_hash, user_id, expires_at) VALUES ($1, $2, $3)`,
		tokenHash, userID, expiresAt)
	return err
}

func (s *Store) SessionUser(ctx context.Context, tokenHash []byte, now time.Time) (domain.User, error) {
	var u domain.User
	err := s.pool.QueryRow(ctx, `SELECT u.id::text, u.name, u.time_zone FROM auth_sessions a JOIN users u ON u.id = a.user_id
		WHERE a.token_hash = $1 AND a.expires_at > $2`, tokenHash, now).Scan(&u.ID, &u.Name, &u.TimeZone)
	return u, nf(err)
}

func (s *Store) DeleteSession(ctx context.Context, tokenHash []byte) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM auth_sessions WHERE token_hash = $1`, tokenHash)
	return err
}
