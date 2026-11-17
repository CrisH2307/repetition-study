package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/CrisH2307/repetition-study/internal/domain"
	"github.com/CrisH2307/repetition-study/internal/fakes"
)

func TestWriteDisputes(t *testing.T) {
	rs := &fakes.ReviewStore{Sessions: map[string]string{"s1": "u1"}}
	rs.CreateDispute(context.Background(), "u1", domain.Dispute{SessionID: "s1", Note: "=HYPERLINK(\"x\")"})
	rs.CreateDispute(context.Background(), "u1", domain.Dispute{SessionID: "s1", Note: "a, \"quoted\" note"})
	var b bytes.Buffer
	if err := writeDisputes(context.Background(), rs, &b); err != nil {
		t.Fatal(err)
	}
	got := b.String()
	if !strings.HasPrefix(got, "created_at,dispute_id,user_id,session_id,claim_id,status,note\n") {
		t.Errorf("header: %q", got)
	}
	if strings.Contains(got, ",=HYPERLINK") || !strings.Contains(got, "'=HYPERLINK") {
		t.Errorf("a note starting with = must be defused: %q", got)
	}
	if !strings.Contains(got, `"a, ""quoted"" note"`) {
		t.Errorf("csv quoting: %q", got)
	}
}

func TestWriteMetrics(t *testing.T) {
	fm := &fakes.MetricsStore{Users: []domain.UserWeeks{{UserID: "real-id"}}}
	var b bytes.Buffer
	now := time.Date(2026, 11, 16, 0, 0, 0, 0, time.UTC)
	if err := writeMetrics(context.Background(), fm, 4, now, "s", &b); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(b.String(), "real-id") || !strings.Contains(b.String(), "week4_retention") {
		t.Fatal(b.String())
	}
	if writeMetrics(context.Background(), fm, 4, now, "", &b) == nil || writeMetrics(context.Background(), 1, 4, now, "s", &b) == nil {
		t.Fatal("want errors for no salt and unsupported store")
	}
}

// metricsAndSpend is what the Postgres store is: metrics and a ledger in one.
type metricsAndSpend struct {
	*fakes.MetricsStore
	*fakes.SpendLedger
}

func TestWriteMetricsCostPerCheckedSession(t *testing.T) {
	now := time.Date(2026, 11, 16, 0, 0, 0, 0, time.UTC)
	src := metricsAndSpend{
		&fakes.MetricsStore{Weeks: []domain.WeekCounts{{WeekStart: time.Date(2026, 11, 9, 0, 0, 0, 0, time.UTC), Sessions: 4, CheckedSessions: 4}}},
		&fakes.SpendLedger{},
	}
	src.SpendLedger.Add(context.Background(), now, 2)
	var b bytes.Buffer
	if err := writeMetrics(context.Background(), src, 2, now, "s", &b); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(b.String(), "2026-11,2.0000,4,0.5000") {
		t.Fatal(b.String())
	}
}
