// Command migrate applies migrations/*.sql in name order, once each.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sort"

	"github.com/jackc/pgx/v5"

	"github.com/CrisH2307/repetition-study/migrations"
)

func main() {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		log.Fatal("DATABASE_URL is not set")
	}
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, url)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close(ctx)

	if _, err := conn.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		name text PRIMARY KEY, applied_at timestamptz NOT NULL DEFAULT now())`); err != nil {
		log.Fatal(err)
	}
	files, err := migrations.FS.ReadDir(".")
	if err != nil {
		log.Fatal(err)
	}
	names := make([]string, 0, len(files))
	for _, f := range files {
		if !f.IsDir() && len(f.Name()) > 4 && f.Name()[len(f.Name())-4:] == ".sql" {
			names = append(names, f.Name())
		}
	}
	sort.Strings(names)

	// ponytail: no lock, so two concurrent runs can race. Fine for one dev machine.
	for _, name := range names {
		var done bool
		if err := conn.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE name = $1)`, name).Scan(&done); err != nil {
			log.Fatal(err)
		}
		if done {
			continue
		}
		sql, err := migrations.FS.ReadFile(name)
		if err != nil {
			log.Fatal(err)
		}
		if err := apply(ctx, conn, name, string(sql)); err != nil {
			log.Fatalf("%s: %v", name, err)
		}
		fmt.Println("applied", name)
	}
}

func apply(ctx context.Context, conn *pgx.Conn, name, sql string) error {
	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, sql); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (name) VALUES ($1)`, name); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
