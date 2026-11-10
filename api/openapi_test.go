package api_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/getkin/kin-openapi/openapi3"

	"github.com/CrisH2307/repetition-study/internal/domain"
)

func load(t *testing.T) *openapi3.T {
	t.Helper()
	doc, err := openapi3.NewLoader().LoadFromFile("openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if err := doc.Validate(context.Background()); err != nil {
		t.Fatalf("openapi.yaml is not a valid OpenAPI document: %v", err)
	}
	return doc
}

func TestSpecIsValid(t *testing.T) { load(t) }

// The spec must accept what the domain types actually marshal to, so the
// spec and internal/domain cannot drift apart unnoticed.
func TestDomainJSONMatchesSpec(t *testing.T) {
	doc := load(t)
	str := func(s string) *string { return &s }
	g := domain.GradeAgain
	now := time.Date(2026, 11, 9, 9, 0, 0, 0, time.UTC)
	ev := domain.ReviewEvent{
		ID: "e1", EventType: domain.EventTeachBack, ConceptID: "c1", Grade: &g,
		DueBefore: now, DueAfter: now, Payload: json.RawMessage(`{}`), CreatedAt: now,
		SessionID: str("s1"), ClaimID: str("cl1"), SelfGrade: &g,
		CheckerModel: str("m"), CheckerVersion: str("v"),
	}
	cases := map[string]any{
		"ReviewEvent":       ev,
		"ReviewEvent nulls": domain.ReviewEvent{ID: "e2", EventType: domain.EventCardReview, ConceptID: "c1", Payload: json.RawMessage(`{}`)},
		"Report":            domain.Report{SessionID: "s1", ConceptID: "c1", Results: []domain.ClaimResult{}, Schedule: &ev, Status: domain.ReportChecked},
		"Report unchecked":  domain.Report{SessionID: "s1", ConceptID: "c1", Results: []domain.ClaimResult{}, Status: domain.ReportUnchecked},
		"DueItem":           domain.DueItem{ConceptID: "c1", ConceptName: "n", DueAt: now, Reason: "due for review"},
	}
	for name, v := range cases {
		schema, _, _ := strings.Cut(name, " ")
		raw, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		var data any
		if err := json.Unmarshal(raw, &data); err != nil {
			t.Fatal(err)
		}
		sv := doc.Components.Schemas[schema].Value
		if err := doc.ValidateSchemaJSON(sv, data); err != nil {
			t.Errorf("%s: domain JSON does not match the spec: %v\n%s", name, err, raw)
		}
		for key := range data.(map[string]any) {
			if sv.Properties[key] == nil {
				t.Errorf("%s: field %q is in internal/domain but not in the spec", name, key)
			}
		}
	}

	// A status outside the enum must be rejected, or the check above proves nothing.
	bad := map[string]any{"session_id": "s", "concept_id": "c", "results": []any{}, "schedule": nil, "status": "maybe"}
	if err := doc.ValidateSchemaJSON(doc.Components.Schemas["Report"].Value, bad); err == nil {
		t.Error("spec accepted status \"maybe\"")
	}
}
