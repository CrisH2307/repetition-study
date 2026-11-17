package fakes_test

import (
	"context"
	"errors"
	"testing"

	"github.com/CrisH2307/repetition-study/internal/domain"
	"github.com/CrisH2307/repetition-study/internal/fakes"
)

func TestBillingFake(t *testing.T) {
	ctx := context.Background()
	b := &fakes.Billing{Store: &fakes.BillingStore{}}
	if ok, _ := b.Entitled(ctx, "u1"); !ok {
		t.Fatal("billing off must entitle everyone")
	}
	b.Enabled = true
	if ok, _ := b.Entitled(ctx, "u1"); ok {
		t.Fatal("unpaid user entitled")
	}
	if err := b.HandleWebhook(ctx, []byte(`{"id":"ev1"}`), "forged"); !errors.Is(err, domain.ErrBadSignature) {
		t.Fatalf("bad signature: %v", err)
	}
	body := []byte(`{"id":"ev1","type":"checkout.session.completed","user_id":"u1","plan":"month"}`)
	if err := b.HandleWebhook(ctx, body, "valid"); err != nil {
		t.Fatal(err)
	}
	if ok, _ := b.Entitled(ctx, "u1"); !ok {
		t.Fatal("paid user not entitled")
	}
	applied, err := b.Store.ApplyEvent(ctx, domain.BillingEvent{EventID: "ev1"}, "u1", nil)
	if err != nil || applied {
		t.Fatalf("replayed event applied again: %v %v", applied, err)
	}
}

func TestSharingFake(t *testing.T) {
	ctx := context.Background()
	pc := domain.PackClaim{Text: "t", Kind: domain.KindDefinition}
	s := &fakes.Sharing{Store: &fakes.ShareStore{},
		Packs: map[string]domain.PackPreview{"sub1": {CourseName: "Stats", Concepts: []domain.PackConcept{{Name: "n", Claims: []domain.PackClaim{pc}}}}}}
	var ue *domain.UserError
	if _, err := s.Create(ctx, "u1", "sub1", true); !errors.As(err, &ue) {
		t.Fatalf("non-instructor shared notes: %v", err)
	}
	ns, err := s.Create(ctx, "u1", "sub1", false)
	if err != nil {
		t.Fatal(err)
	}
	imp, err := s.Import(ctx, "u2", ns.Token, "sub9")
	if err != nil || imp.ConceptsAdded != 1 || len(imp.NeedsQuote) != 1 {
		t.Fatalf("import: %+v %v", imp, err)
	}
	if err := s.Revoke(ctx, "u2", ns.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("other user revoked: %v", err)
	}
	if err := s.Revoke(ctx, "u1", ns.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Preview(ctx, ns.Token); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("revoked link still works: %v", err)
	}
}
