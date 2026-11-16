package main

import (
	"bytes"
	"context"
	"strings"
	"testing"

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
