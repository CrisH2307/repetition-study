package main

import (
	"bytes"
	"context"
	"testing"

	"github.com/CrisH2307/repetition-study/internal/fakes"
)

func TestRunSetsAndClearsFlag(t *testing.T) {
	st := &fakes.ShareStore{}
	var out bytes.Buffer
	if err := run(context.Background(), st, "u1", true, &out); err != nil || !st.Instructors["u1"] {
		t.Fatalf("on: %v %v", err, st.Instructors)
	}
	if err := run(context.Background(), st, "u1", false, &out); err != nil || st.Instructors["u1"] {
		t.Fatalf("off: %v %v", err, st.Instructors)
	}
	if run(context.Background(), st, "", true, &out) == nil {
		t.Fatal("empty user must fail")
	}
}
