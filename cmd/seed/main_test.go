package main

import (
	"context"
	"strings"
	"testing"

	"github.com/CrisH2307/repetition-study/internal/fakes"
)

func TestFixtureLoads(t *testing.T) {
	p, err := readPack("../../content/fixture/unit.json")
	if err != nil {
		t.Fatal(err)
	}
	l := &fakes.PackLoader{}
	if err := load(context.Background(), l, p); err != nil {
		t.Fatal(err)
	}
	if len(l.Packs) != 1 || len(p.Concepts) == 0 {
		t.Fatalf("loaded %d packs, %d concepts", len(l.Packs), len(p.Concepts))
	}
}

// The committed unit.json has no course text (rule: course material is never
// committed), so seeding it alone must fail loudly instead of loading a source
// the checker could never quote.
func TestEmptySourceTextRefused(t *testing.T) {
	p, err := readPack("../../content/unit.json")
	if err != nil {
		t.Fatal(err)
	}
	l := &fakes.PackLoader{}
	err = load(context.Background(), l, p)
	if err == nil || !strings.Contains(err.Error(), "no text") || len(l.Packs) != 0 {
		t.Fatalf("err = %v, packs loaded = %d", err, len(l.Packs))
	}
}
