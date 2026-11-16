package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/CrisH2307/repetition-study/internal/domain"
)

// disputeRow is one line of the file scripts/disputes_to_eval.py writes.
type disputeRow struct {
	ID        string            `json:"id"`
	ConceptID string            `json:"concept_id"`
	ClaimID   string            `json:"claim_id"`
	Text      string            `json:"text"`
	Expected  domain.ClaimLabel `json:"expected"`
}

// addDisputes adds upheld disputes (D39) to the loaded data as extra
// explanations, each labelled for its one disputed claim only. Dev split only:
// the gate is decided on the frozen test split, which this never touches.
func addDisputes(d *dataset, path string) error {
	if d.Split == "test" {
		return errors.New("--disputes adds development cases only: use --split dev (or all for a look), never test")
	}
	b, why, err := d.read(path)
	if err != nil {
		return err
	}
	if why != "" {
		return errors.New(why)
	}
	rows, err := jsonl[disputeRow](path, b)
	if err != nil {
		return err
	}
	if d.Labels == nil {
		d.Labels = map[labelKey]domain.ClaimLabel{}
	}
	for _, r := range rows {
		c, ok := d.Concepts[r.ConceptID]
		if !ok {
			return fmt.Errorf("%s: %s names concept %q, which is not in the unit", path, r.ID, r.ConceptID)
		}
		if !strings.HasPrefix(r.ID, "d-") {
			return fmt.Errorf("%s: id %q must start with d-, so it cannot clash with a frozen case", path, r.ID)
		}
		if _, dup := d.AllExpl[r.ID]; dup {
			return fmt.Errorf("%s: id %q appears twice", path, r.ID)
		}
		found := false
		for _, k := range c.Claims {
			found = found || k.ID == r.ClaimID
		}
		if !found {
			return fmt.Errorf("%s: %s names claim %q, which is not in its concept", path, r.ID, r.ClaimID)
		}
		switch r.Expected {
		case domain.LabelSupported, domain.LabelPartial, domain.LabelContradicted, domain.LabelAbsent:
		default:
			return fmt.Errorf("%s: %s has expected label %q", path, r.ID, r.Expected)
		}
		e := explanation{ID: r.ID, ConceptID: r.ConceptID, Text: r.Text}
		d.AllExpl[e.ID] = e
		d.Expl = append(d.Expl, e)
		d.Labels[labelKey{e.ID, r.ClaimID}] = r.Expected
	}
	if len(rows) > 0 {
		d.ExplWhy, d.LabelsWhy = "", ""
	}
	return nil
}
