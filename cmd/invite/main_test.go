package main

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/CrisH2307/repetition-study/internal/auth"
	"github.com/CrisH2307/repetition-study/internal/fakes"
)

func TestRunPrintsRedeemableLink(t *testing.T) {
	a := &auth.Service{Store: &fakes.AuthStore{}, Clock: fakes.NewClock()}
	var out bytes.Buffer
	if err := run(context.Background(), a, " Dino ", "https://pilot.example/", &out); err != nil {
		t.Fatal(err)
	}
	link := strings.TrimSpace(out.String())
	tok, ok := strings.CutPrefix(link, "https://pilot.example/invite#")
	if !ok {
		t.Fatalf("link %q", link)
	}
	u, _, _, err := a.Redeem(context.Background(), tok)
	if err != nil || u.Name != "Dino" {
		t.Fatalf("redeem: %v %+v", err, u)
	}
	if err := run(context.Background(), a, "  ", "x", &out); err == nil {
		t.Fatal("blank name accepted")
	}
}
