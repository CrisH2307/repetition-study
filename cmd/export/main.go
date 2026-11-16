// Command export writes data for people to read.
//
//	DATABASE_URL=... go run ./cmd/export -disputes > disputes.csv
//
// -disputes lists every "I think this check is wrong" note, oldest first, so
// Dino can review them. It reads only; nothing changes.
package main

import (
	"context"
	"encoding/csv"
	"flag"
	"io"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/CrisH2307/repetition-study/internal/domain"
	"github.com/CrisH2307/repetition-study/internal/store"
)

func main() {
	disputes := flag.Bool("disputes", false, "write every dispute as CSV")
	flag.Parse()
	if !*disputes {
		log.Fatal("nothing to export: use -disputes")
	}
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
	if err := writeDisputes(ctx, store.New(pool), os.Stdout); err != nil {
		log.Fatal(err)
	}
}

func writeDisputes(ctx context.Context, s domain.ReviewStore, out io.Writer) error {
	ds, err := s.AllDisputes(ctx)
	if err != nil {
		return err
	}
	w := csv.NewWriter(out)
	w.Write([]string{"created_at", "dispute_id", "user_id", "session_id", "claim_id", "status", "note"})
	for _, d := range ds {
		claim := ""
		if d.ClaimID != nil {
			claim = *d.ClaimID
		}
		w.Write([]string{d.CreatedAt.UTC().Format(time.RFC3339), d.ID, d.UserID, d.SessionID, claim, string(d.Status), safe(d.Note)})
	}
	w.Flush()
	return w.Error()
}

// safe stops a spreadsheet from running a student's note as a formula.
func safe(s string) string {
	if s != "" && (s[0] == '=' || s[0] == '+' || s[0] == '-' || s[0] == '@' || s[0] == '\t' || s[0] == '\r') {
		return "'" + s
	}
	return s
}
