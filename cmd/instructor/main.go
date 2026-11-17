// Command instructor marks a user as an instructor, or removes the mark (D51).
// Only Cris runs it; there is no self-serve path.
//
//	DATABASE_URL=... go run ./cmd/instructor -user <user id>
//	DATABASE_URL=... go run ./cmd/instructor -user <user id> -off
//
// Users have no email column, so the user id is the key (see `users` in the db).
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/CrisH2307/repetition-study/internal/domain"
	"github.com/CrisH2307/repetition-study/internal/store"
)

func main() {
	user := flag.String("user", "", "user id (required)")
	off := flag.Bool("off", false, "remove the instructor mark")
	flag.Parse()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		log.Fatal("DATABASE_URL is not set")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	if err := run(ctx, store.New(pool), *user, !*off, os.Stdout); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, st domain.ShareStore, user string, on bool, out io.Writer) error {
	if user == "" {
		return fmt.Errorf("-user is required")
	}
	if err := st.SetInstructor(ctx, user, on); err != nil {
		return err
	}
	_, err := fmt.Fprintf(out, "user %s instructor = %v\n", user, on)
	return err
}
