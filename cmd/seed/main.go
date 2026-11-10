// Command seed loads a pack into the database through domain.PackLoader.
// Without -pack it loads content/private/unit.json (real source text) when that
// file exists, else the committed fixture content/fixture/unit.json.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/CrisH2307/repetition-study/internal/domain"
	"github.com/CrisH2307/repetition-study/internal/store"
)

const (
	privatePack = "content/private/unit.json"
	fixturePack = "content/fixture/unit.json"
)

func main() {
	path := flag.String("pack", "", "pack file (default: "+privatePack+" if present, else "+fixturePack+")")
	flag.Parse()
	if *path == "" {
		*path = fixturePack
		if _, err := os.Stat(privatePack); err == nil {
			*path = privatePack
		}
	}
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		log.Fatal("DATABASE_URL is not set")
	}
	pack, err := readPack(*path)
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	if err := load(ctx, store.New(pool), pack); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("seeded %s from %s: %d sources, %d concepts\n", pack.Subject.Name, *path, len(pack.Sources), len(pack.Concepts))
}

func readPack(path string) (domain.Pack, error) {
	var p domain.Pack
	b, err := os.ReadFile(path)
	if err != nil {
		return p, err
	}
	if err := json.Unmarshal(b, &p); err != nil {
		return p, fmt.Errorf("%s: %w", path, err)
	}
	return p, nil
}

// load refuses a pack whose course source has no text: the checker could never
// quote it, so every correction would be dropped.
func load(ctx context.Context, l domain.PackLoader, p domain.Pack) error {
	for _, s := range p.Sources {
		if s.Kind == domain.SourceCourse && s.Text == "" {
			return fmt.Errorf("source %q (%s) has no text: put the real unit in content/private/unit.json or use the fixture", s.ID, s.Title)
		}
	}
	return l.LoadPack(ctx, p)
}
