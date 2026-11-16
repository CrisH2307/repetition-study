// Package fakes holds deterministic in-memory implementations of every domain
// interface, so units can build and test alone. No randomness, no real clock.
package fakes

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/CrisH2307/repetition-study/internal/domain"
)

var (
	_ domain.LLM         = (*LLM)(nil)
	_ domain.TeachEngine = (*TeachEngine)(nil)
	_ domain.Checker     = (*Checker)(nil)
	_ domain.Scheduler   = (*Scheduler)(nil)
	_ domain.Planner     = (*Planner)(nil)
	_ domain.Digester    = (*Digester)(nil)
	_ domain.Clock       = (*Clock)(nil)

	_ domain.ReportReader     = (*TeachEngine)(nil)
	_ domain.SessionScheduler = (*Scheduler)(nil)
	_ domain.TeachStore       = (*TeachStore)(nil)
	_ domain.PackLoader       = (*PackLoader)(nil)
)

// Epoch is the fixed start time for FakeClock: Monday 2026-11-09 09:00 UTC.
var Epoch = time.Date(2026, 11, 9, 9, 0, 0, 0, time.UTC)

// Clock returns a fixed time that only moves when Advance is called.
type Clock struct {
	mu sync.Mutex
	t  time.Time
}

func NewClock() *Clock { return &Clock{t: Epoch} }

func (c *Clock) Now() time.Time { c.mu.Lock(); defer c.mu.Unlock(); return c.t }

func (c *Clock) Advance(d time.Duration) { c.mu.Lock(); c.t = c.t.Add(d); c.mu.Unlock() }

// LLM replays fixed text. Stream emits Reply one word per delta. JSON decodes
// JSONReply into out. Requests records every call so tests can assert on what
// a model was (or was not) sent.
type LLM struct {
	Reply     string
	JSONReply string

	mu       sync.Mutex
	Requests []domain.LLMRequest
}

func (l *LLM) record(r domain.LLMRequest) {
	l.mu.Lock()
	l.Requests = append(l.Requests, r)
	l.mu.Unlock()
}

func (l *LLM) Stream(ctx context.Context, req domain.LLMRequest) (<-chan domain.Delta, error) {
	l.record(req)
	reply := l.Reply
	if reply == "" {
		reply = "What do you mean by that?"
	}
	words := strings.Fields(reply)
	ch := make(chan domain.Delta, len(words))
	for i, w := range words {
		if i > 0 {
			w = " " + w
		}
		ch <- domain.Delta{Text: w}
	}
	close(ch)
	return ch, ctx.Err()
}

func (l *LLM) JSON(ctx context.Context, req domain.LLMRequest, out any) error {
	l.record(req)
	if err := ctx.Err(); err != nil {
		return err
	}
	reply := l.JSONReply
	if reply == "" {
		reply = "{}"
	}
	return json.Unmarshal([]byte(reply), out)
}

// TeachEngine hands out sessions sess-1, sess-2, ... and asks canned questions.
type TeachEngine struct {
	mu       sync.Mutex
	n        int
	sessions map[string]string // session id -> concept id
	reports  map[string]domain.Report
	Report   *domain.Report // optional canned Finish result
}

func (t *TeachEngine) Start(_ context.Context, conceptID string) (domain.TeachSession, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.n++
	if t.sessions == nil {
		t.sessions = map[string]string{}
	}
	id := fmt.Sprintf("sess-%d", t.n)
	t.sessions[id] = conceptID
	return domain.TeachSession{ID: id, ConceptID: conceptID}, nil
}

func (t *TeachEngine) Turn(ctx context.Context, sessionID, transcript, latex string) (<-chan domain.Delta, error) {
	t.mu.Lock()
	_, ok := t.sessions[sessionID]
	t.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("fakes: unknown session %q", sessionID)
	}
	l := &LLM{Reply: "Can you say more about that?"}
	return l.Stream(ctx, domain.LLMRequest{Messages: []domain.Message{{Role: "user", Content: transcript + latex}}})
}

func (t *TeachEngine) Finish(_ context.Context, sessionID string) (domain.Report, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	concept, ok := t.sessions[sessionID]
	if !ok {
		return domain.Report{}, fmt.Errorf("fakes: unknown session %q", sessionID)
	}
	r := domain.Report{SessionID: sessionID, ConceptID: concept, Status: domain.ReportChecked}
	if t.Report != nil {
		r = *t.Report
	}
	if t.reports == nil {
		t.reports = map[string]domain.Report{}
	}
	t.reports[sessionID] = r
	return r, nil
}

// GetReport returns what Finish returned for the session.
func (t *TeachEngine) GetReport(_ context.Context, sessionID string) (domain.Report, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	r, ok := t.reports[sessionID]
	if !ok {
		return domain.Report{}, fmt.Errorf("fakes: no report for session %q", sessionID)
	}
	return r, nil
}

// Checker returns Results if set, else one supported result per required claim.
type Checker struct {
	Results []domain.ClaimResult
}

func (c *Checker) Check(_ context.Context, in domain.CheckInput) ([]domain.ClaimResult, error) {
	if c.Results != nil {
		return c.Results, nil
	}
	var out []domain.ClaimResult
	for _, cl := range in.Concept.Claims {
		if !cl.Required {
			continue
		}
		id := cl.ID
		out = append(out, domain.ClaimResult{ClaimID: &id, StudentText: cl.Text, Label: domain.LabelSupported})
	}
	return out, nil
}

// Scheduler keeps due dates in memory. Fixed intervals per grade stand in for FSRS.
type Scheduler struct {
	Clock domain.Clock

	mu     sync.Mutex
	n      int
	due    map[string]time.Time
	Events []domain.ReviewEvent
}

var gradeDays = map[domain.Grade]int{domain.GradeAgain: 1, domain.GradeHard: 3, domain.GradeGood: 7, domain.GradeEasy: 14}

func (s *Scheduler) now() time.Time {
	if s.Clock == nil {
		return Epoch
	}
	return s.Clock.Now()
}

// event records one append-only event. Caller holds s.mu.
func (s *Scheduler) event(et domain.EventType, concept string, card *string, g *domain.Grade, after time.Time, payload string) domain.ReviewEvent {
	if s.due == nil {
		s.due = map[string]time.Time{}
	}
	before, ok := s.due[concept]
	if !ok {
		before = s.now()
	}
	s.due[concept] = after
	s.n++
	ev := domain.ReviewEvent{
		ID: fmt.Sprintf("evt-%d", s.n), EventType: et, ConceptID: concept, CardID: card, Grade: g,
		DueBefore: before, DueAfter: after, Payload: json.RawMessage(payload), CreatedAt: s.now(),
	}
	s.Events = append(s.Events, ev)
	return ev
}

func (s *Scheduler) ApplyTeachBack(_ context.Context, conceptID string, results []domain.ClaimResult) (domain.ReviewEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.teachBack(conceptID, results), nil
}

// ApplyTeachBackSession is ApplyTeachBack keyed by session: a repeat call for
// the same session returns the first event and appends nothing.
func (s *Scheduler) ApplyTeachBackSession(_ context.Context, sessionID, conceptID string, results []domain.ClaimResult) (domain.ReviewEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, ev := range s.Events {
		if ev.EventType == domain.EventTeachBack && ev.SessionID != nil && *ev.SessionID == sessionID {
			return ev, nil
		}
	}
	ev := s.teachBack(conceptID, results)
	ev.SessionID = &sessionID
	s.Events[len(s.Events)-1] = ev
	return ev, nil
}

// teachBack grades results and records the event. Caller holds s.mu.
func (s *Scheduler) teachBack(conceptID string, results []domain.ClaimResult) domain.ReviewEvent {
	g := domain.GradeGood // plan: contradicted or absent -> again, partial -> hard, else good, never easy
	for _, r := range results {
		switch r.Label {
		case domain.LabelContradicted, domain.LabelAbsent:
			g = domain.GradeAgain
		case domain.LabelPartial:
			if g != domain.GradeAgain {
				g = domain.GradeHard
			}
		}
	}
	return s.event(domain.EventTeachBack, conceptID, nil, &g, s.now().AddDate(0, 0, gradeDays[g]), "{}")
}

func (s *Scheduler) ReviewCard(_ context.Context, cardID string, grade domain.Grade) (domain.ReviewEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.event(domain.EventCardReview, "concept-1", &cardID, &grade, s.now().AddDate(0, 0, gradeDays[grade]), "{}"), nil
}

func (s *Scheduler) Override(_ context.Context, conceptID string, dueAt time.Time, reason string) (domain.ReviewEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, _ := json.Marshal(map[string]string{"reason": reason})
	return s.event(domain.EventManualOverride, conceptID, nil, nil, dueAt, string(p)), nil
}

func (s *Scheduler) Due(_ context.Context, now time.Time) ([]domain.DueItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []domain.DueItem
	for c, d := range s.due {
		if !d.After(now) {
			out = append(out, domain.DueItem{ConceptID: c, DueAt: d, Reason: "due for review"})
		}
	}
	// Map order is random, so sort to stay deterministic.
	sort.Slice(out, func(i, j int) bool { return out[i].ConceptID < out[j].ConceptID })
	return out, nil
}

// TeachStore keeps sessions, turns and results in memory. IDs are sess-N and
// taught-N. Sources holds the approved sources LoadConcept returns.
type TeachStore struct {
	Concepts map[string]domain.Concept
	Sources  []domain.Source

	mu        sync.Mutex
	n         int
	Sessions  map[string]domain.TeachSession
	Taught    map[string][]domain.TaughtClaim
	Assistant map[string]map[int]string
	Results   map[string][]domain.ClaimResult
	Finished  map[string]bool
}

func (m *TeachStore) init() {
	if m.Sessions == nil {
		m.Sessions, m.Taught = map[string]domain.TeachSession{}, map[string][]domain.TaughtClaim{}
		m.Assistant, m.Results, m.Finished = map[string]map[int]string{}, map[string][]domain.ClaimResult{}, map[string]bool{}
	}
}

func (m *TeachStore) LoadConcept(_ context.Context, id string) (domain.Concept, []domain.Source, error) {
	c, ok := m.Concepts[id]
	if !ok {
		return domain.Concept{}, nil, fmt.Errorf("fakes: unknown concept %q", id)
	}
	return c, m.Sources, nil
}

func (m *TeachStore) StartSession(_ context.Context, conceptID string) (domain.TeachSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.init()
	m.n++
	s := domain.TeachSession{ID: fmt.Sprintf("sess-%d", m.n), ConceptID: conceptID}
	m.Sessions[s.ID] = s
	return s, nil
}

func (m *TeachStore) SaveTurn(_ context.Context, t domain.TaughtClaim, _, _ string) (domain.TaughtClaim, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.init()
	if _, ok := m.Sessions[t.SessionID]; !ok {
		return domain.TaughtClaim{}, fmt.Errorf("fakes: unknown session %q", t.SessionID)
	}
	m.n++
	t.ID = fmt.Sprintf("taught-%d", m.n)
	m.Taught[t.SessionID] = append(m.Taught[t.SessionID], t)
	return t, nil
}

func (m *TeachStore) SetAssistantText(_ context.Context, sid string, turnNo int, text string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.init()
	if m.Assistant[sid] == nil {
		m.Assistant[sid] = map[int]string{}
	}
	m.Assistant[sid][turnNo] = text
	return nil
}

func (m *TeachStore) SaveResults(_ context.Context, sid string, r []domain.ClaimResult) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.init()
	m.Results[sid] = r
	return nil
}

func (m *TeachStore) FinishSession(_ context.Context, sid string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.init()
	m.Finished[sid] = true
	return nil
}

// PackLoader records every pack it is given.
type PackLoader struct {
	mu    sync.Mutex
	Packs []domain.Pack
}

func (p *PackLoader) LoadPack(_ context.Context, pack domain.Pack) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Packs = append(p.Packs, pack)
	return nil
}

// Planner returns one fixed 30 minute review block per call.
type Planner struct{}

func (Planner) Generate(_ context.Context, _ string, weekStart time.Time) (domain.Plan, error) {
	return domain.Plan{WeekStart: weekStart, Blocks: []domain.PlanBlock{{
		ID: "block-1", SubjectID: "subject-1", Kind: domain.PlanReview,
		StartsAt: weekStart.Add(9 * time.Hour), Minutes: 30, Reason: "fake plan",
	}}}, nil
}

func (Planner) FurtherPlanning(_ context.Context, weekStart time.Time) (domain.FurtherPlanning, error) {
	return domain.FurtherPlanning{WeekStart: weekStart, Subjects: []domain.SubjectProgress{{
		SubjectID: "subject-1", Name: "Fake Subject", NextWeekHours: 3,
		Concepts: []domain.ConceptProgress{{ConceptID: "concept-1", Name: "Fake Concept"}},
	}}}, nil
}

// Digester returns one fixed subject line.
type Digester struct{}

func (Digester) Today(_ context.Context, now time.Time) (domain.Digest, error) {
	day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	return domain.Digest{Date: day, Items: []domain.DigestItem{
		{SubjectID: "subject-1", SubjectName: "Fake Subject", DueCount: 2, Minutes: 15},
	}}, nil
}
