package fakes

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/CrisH2307/repetition-study/internal/domain"
)

func TestImportFakeScopesAndCountsRows(t *testing.T) {
	ctx := context.Background()
	st := &ImportStore{Owners: map[string]string{"sub-1": "u1"}}
	im := &Importer{Store: st}
	j, err := im.Import(ctx, "u1", "sub-1", "c.csv", []byte("concept,claim,kind,quote\nA,B,definition,Q\nA,,statement,Q\n"))
	if err != nil || j.Status != domain.ImportDone || j.RowsTotal != 2 || j.RowsImported != 1 || len(j.RowErrors) != 1 || j.RowErrors[0].Row != 3 {
		t.Fatalf("%+v %v", j, err)
	}
	if _, err := st.GetImport(ctx, "u2", j.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatal("another user read the import")
	}
	if _, err := im.Import(ctx, "u2", "sub-1", "c.csv", nil); !errors.Is(err, domain.ErrNotFound) {
		t.Fatal("another user imported into the course")
	}
}

func TestEvalCaseFakeNeedsLabelOnUphold(t *testing.T) {
	ctx := context.Background()
	e := &EvalCaseStore{Disputes: map[string]domain.DisputeCase{"d1": {DisputeID: "d1"}, "d2": {DisputeID: "d2"}}}
	if err := e.ResolveDispute(ctx, "d1", domain.DisputeUpheld, nil); err == nil {
		t.Fatal("upheld without a label accepted")
	}
	l := domain.LabelContradicted
	if err := e.ResolveDispute(ctx, "d1", domain.DisputeUpheld, &l); err != nil {
		t.Fatal(err)
	}
	if err := e.ResolveDispute(ctx, "d2", domain.DisputeRejected, nil); err != nil {
		t.Fatal(err)
	}
	got, _ := e.UpheldDisputes(ctx)
	if len(got) != 1 || got[0].ExpectedLabel != l {
		t.Fatalf("%+v", got)
	}
}

func TestSpendLedgerFakeIsPerMonth(t *testing.T) {
	ctx := context.Background()
	var s SpendLedger
	nov := time.Date(2026, 11, 30, 23, 0, 0, 0, time.UTC)
	s.Add(ctx, nov, 1.5)
	if tot, _ := s.Add(ctx, nov, 0.25); tot != 1.75 {
		t.Fatalf("total %v", tot)
	}
	if tot, _ := s.MonthTotal(ctx, nov.Add(2*time.Hour)); tot != 0 {
		t.Fatalf("December started at %v", tot)
	}
	if _, err := s.Add(ctx, nov, -1); err == nil {
		t.Fatal("negative spend accepted")
	}
}

func TestPrivacyFakeDeletes(t *testing.T) {
	ctx := context.Background()
	p := &PrivacyStore{Exports: map[string]domain.UserExport{"u1": {User: domain.User{ID: "u1"}}}}
	if _, err := p.ExportUser(ctx, "u1", Epoch); err != nil {
		t.Fatal(err)
	}
	if err := p.DeleteUser(ctx, "u1"); err != nil {
		t.Fatal(err)
	}
	if _, err := p.ExportUser(ctx, "u1", Epoch); !errors.Is(err, domain.ErrNotFound) {
		t.Fatal("deleted user still exports")
	}
}
