package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/CrisH2307/repetition-study/internal/domain"
)

// q is what both the pool and a transaction offer.
type q interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

var (
	_ domain.AuthStore   = (*Store)(nil)
	_ domain.CourseStore = (*Store)(nil)
	_ domain.ReviewStore = (*Store)(nil)
)

// pgCode returns the Postgres error code, or "".
func pgCode(err error) string {
	var pe *pgconn.PgError
	if errors.As(err, &pe) {
		return pe.Code
	}
	return ""
}

// nf turns "no row" and "not a uuid" into domain.ErrNotFound, so a made-up id
// looks the same as another user's id.
func nf(err error) error {
	if errors.Is(err, pgx.ErrNoRows) || pgCode(err) == "22P02" {
		return domain.ErrNotFound
	}
	return err
}

// affected is ErrNotFound when the statement matched no row.
func affected(tag pgconn.CommandTag, err error) error {
	if err != nil {
		return nf(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func collect[T any](rows pgx.Rows, err error, scan func(pgx.Row) (T, error)) ([]T, error) {
	if err != nil {
		return nil, nf(err)
	}
	out, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (T, error) { return scan(r) })
	if out == nil {
		out = []T{}
	}
	return out, err
}
