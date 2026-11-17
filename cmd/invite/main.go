// Command invite prints a one-time sign-in link for a new student (D29).
//
//	DATABASE_URL=... APP_URL=https://pilot.example go run ./cmd/invite -name "Dino"
//
// The token sits after "#" so it never reaches server logs or Referer headers;
// the web app's /invite page reads it and posts it to /v1/auth/redeem.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/CrisH2307/repetition-study/internal/auth"
	"github.com/CrisH2307/repetition-study/internal/domain"
	"github.com/CrisH2307/repetition-study/internal/store"
)

func main() {
	name := flag.String("name", "", "student's name (required)")
	revoke := flag.String("revoke", "", "revoke this invite (the link or its token) and end its sessions")
	flag.Parse()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		log.Fatal("DATABASE_URL is not set")
	}
	base := os.Getenv("APP_URL")
	if base == "" {
		base = "http://localhost:5173"
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	// ponytail: runtime check until P1-STORE lands store.AuthStore; then make it a plain conversion.
	st, ok := any(store.New(pool)).(domain.AuthStore)
	if !ok {
		log.Fatal("this build's store has no invites yet (needs P1-STORE)")
	}
	if *revoke != "" {
		if err := revokeInvite(ctx, &auth.Service{Store: st}, *revoke, os.Stdout); err != nil {
			log.Fatal(err)
		}
		return
	}
	if err := run(ctx, &auth.Service{Store: st}, *name, base, os.Stdout); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, a domain.Auth, name, base string, out io.Writer) error {
	name = strings.TrimSpace(name)
	tok, err := a.NewInvite(ctx, name)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(out, "%s/invite#%s\n", strings.TrimRight(base, "/"), tok)
	return err
}

// revokeInvite takes the whole link or just the token after "#".
func revokeInvite(ctx context.Context, a domain.Auth, link string, out io.Writer) error {
	if _, tok, ok := strings.Cut(link, "#"); ok {
		link = tok
	}
	if err := a.RevokeInvite(ctx, strings.TrimSpace(link)); err != nil {
		return err
	}
	_, err := fmt.Fprintln(out, "Invite revoked. Its sign-ins have ended.")
	return err
}
