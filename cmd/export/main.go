// Command export writes data for people to read.
//
//	DATABASE_URL=... go run ./cmd/export -disputes > disputes.csv
//
// -disputes lists every "I think this check is wrong" note, oldest first, so
// Dino can review them. It reads only; nothing changes.
//
//	DATABASE_URL=... EXPORT_SALT=... go run ./cmd/export -metrics [-weeks 12] > metrics.csv
//
// -metrics writes the pilot numbers (D36), plus model cost per checked session by month, with user ids hashed using EXPORT_SALT.
package main

import (
	"context"
	"encoding/csv"
	"errors"
	"flag"
	"io"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/CrisH2307/repetition-study/internal/domain"
	"github.com/CrisH2307/repetition-study/internal/metrics"
	"github.com/CrisH2307/repetition-study/internal/store"
)

func main() {
	disputes := flag.Bool("disputes", false, "write every dispute as CSV")
	metricsOn := flag.Bool("metrics", false, "write pilot metrics as CSV")
	weeks := flag.Int("weeks", 12, "weeks of history for -metrics")
	flag.Parse()
	if !*disputes && !*metricsOn {
		log.Fatal("nothing to export: use -disputes or -metrics")
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
	st := store.New(pool)
	if *metricsOn {
		if err := writeMetrics(ctx, st, *weeks, time.Now(), os.Getenv("EXPORT_SALT"), os.Stdout); err != nil {
			log.Fatal(err)
		}
		return
	}
	if err := writeDisputes(ctx, st, os.Stdout); err != nil {
		log.Fatal(err)
	}
}

// writeMetrics takes any value so the Postgres store can be checked at run time
// (it gains MetricsStore in P2-STORE).
func writeMetrics(ctx context.Context, src any, weeks int, now time.Time, salt string, out io.Writer) error {
	if salt == "" {
		return errors.New("EXPORT_SALT is not set")
	}
	ms, ok := src.(domain.MetricsStore)
	if !ok {
		return errors.New("this store cannot compute metrics yet")
	}
	to := metrics.Monday(now).AddDate(0, 0, 7)
	r, err := metrics.Compute(ctx, ms, to.AddDate(0, 0, -7*weeks), to, now)
	if err != nil {
		return err
	}
	if l, ok := src.(domain.SpendLedger); ok {
		if err := metrics.AddCost(ctx, &r, l); err != nil {
			return err
		}
	}
	if ec, ok := src.(domain.EvalCaseStore); ok {
		up, err := ec.UpheldDisputes(ctx)
		if err != nil {
			return err
		}
		r.Upheld = len(up)
	}
	us, err := ms.UserWeeks(ctx)
	if err != nil {
		return err
	}
	return metrics.WriteCSV(out, r, us, salt)
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
