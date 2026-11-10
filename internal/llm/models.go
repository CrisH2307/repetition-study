package llm

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

//go:embed models.json
var modelsJSON []byte

// ModelInfo is one models.json entry. Ids and prices live there, never in code.
type ModelInfo struct {
	Vendor      string  `json:"vendor"` // "gemini" or "anthropic"; empty means guess from the id prefix
	InPerMTok   float64 `json:"in_per_mtok"`
	OutPerMTok  float64 `json:"out_per_mtok"`
	Source      string  `json:"source"`
	ThinkBudget *int32  `json:"thinking_budget"` // Gemini only
}

// Models is the parsed models.json.
type Models struct {
	Models        map[string]ModelInfo `json:"models"`
	GeminiDefault struct {
		ThinkingBudget int32 `json:"thinking_budget"`
	} `json:"gemini_default"`
}

// LoadModels parses a models.json document.
func LoadModels(b []byte) (Models, error) {
	var m Models
	return m, json.Unmarshal(b, &m)
}

// DefaultModels is the embedded models.json.
func DefaultModels() Models {
	m, err := LoadModels(modelsJSON)
	if err != nil {
		panic("llm: bad embedded models.json: " + err.Error())
	}
	return m
}

// Vendor returns "gemini" or "anthropic" for a model id.
func (m Models) Vendor(id string) (string, error) {
	if v := m.Models[id].Vendor; v != "" {
		return v, nil
	}
	switch {
	case strings.HasPrefix(id, "gemini"):
		return "gemini", nil
	case strings.HasPrefix(id, "claude"):
		return "anthropic", nil
	}
	return "", fmt.Errorf("llm: unknown model %q: add it to models.json", id)
}

// Cost is the USD price of one call. Unlisted models cost 0, see Priced.
func (m Models) Cost(id string, in, out int64) float64 {
	i := m.Models[id]
	return (float64(in)*i.InPerMTok + float64(out)*i.OutPerMTok) / 1e6
}

// Priced reports whether models.json has a price for id.
func (m Models) Priced(id string) bool {
	i, ok := m.Models[id]
	return ok && (i.InPerMTok > 0 || i.OutPerMTok > 0)
}

// Call is one metered model call. Cost is 0 when the model is unpriced.
type Call struct {
	Model   string
	In, Out int64
	Cost    float64
	Latency time.Duration
}

// ErrBudget is returned when a Meter's cap is reached.
var ErrBudget = errors.New("llm: cost cap reached")

// Meter records every call in memory. A nil *Meter is valid and records nothing.
// It is shared by eval and production, so both count cost the same way.
type Meter struct {
	Models Models  // zero value means DefaultModels()
	Cap    float64 // USD; 0 means no cap. Calls fail with ErrBudget once reached.

	mu    sync.Mutex
	calls []Call
}

func (m *Meter) models() Models {
	if m.Models.Models == nil {
		m.Models = DefaultModels()
	}
	return m.Models
}

// Record adds a call and fills in its cost.
func (m *Meter) Record(c Call) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	c.Cost = m.models().Cost(c.Model, c.In, c.Out)
	m.calls = append(m.calls, c)
}

// check fails once spend has reached the cap.
func (m *Meter) check() error {
	if m == nil || m.Cap <= 0 {
		return nil
	}
	if _, _, cost := m.Total(); cost >= m.Cap {
		return fmt.Errorf("%w ($%.2f)", ErrBudget, m.Cap)
	}
	return nil
}

// Calls returns a copy of the recorded calls.
func (m *Meter) Calls() []Call {
	if m == nil {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]Call(nil), m.calls...)
}

// Total sums tokens and cost.
func (m *Meter) Total() (in, out int64, cost float64) {
	for _, c := range m.Calls() {
		in, out, cost = in+c.In, out+c.Out, cost+c.Cost
	}
	return
}

// Summary is one line per session for evalcheck and the server log.
func (m *Meter) Summary() string {
	in, out, cost := m.Total()
	unpriced := map[string]bool{}
	for _, c := range m.Calls() {
		if c.Model != "fake" && !m.models().Priced(c.Model) {
			unpriced[c.Model] = true
		}
	}
	s := fmt.Sprintf("%d calls, %d tokens in, %d out, $%.4f", len(m.Calls()), in, out, cost)
	for id := range unpriced {
		s += fmt.Sprintf(" (no price for %s in models.json)", id)
	}
	return s
}
