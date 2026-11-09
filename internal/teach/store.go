package teach

import (
	"context"
	"fmt"
	"sync"

	"github.com/CrisH2307/repetition-study/internal/domain"
)

// Store is what the engine needs from persistence. The domain package has no
// interface for it. Tables teach_sessions, teach_turns, taught_claims and
// claim_labels (migrations/0001_core.sql) back it. A pgx implementation is
// not part of U2: INT supplies it. MemStore below serves tests and demos.
type Store interface {
	// LoadConcept returns the concept with its required claims and the approved
	// sources those claims cite (course unit and reviewed allowlist only).
	LoadConcept(ctx context.Context, conceptID string) (domain.Concept, []domain.Source, error)
	StartSession(ctx context.Context, conceptID string) (domain.TeachSession, error)
	// SaveTurn stores the turn and its taught claim. The store assigns the claim ID.
	SaveTurn(ctx context.Context, t domain.TaughtClaim, transcript, latex string) (domain.TaughtClaim, error)
	SetAssistantText(ctx context.Context, sessionID string, turnNo int, text string) error
	// SaveResults replaces the session's claim labels.
	SaveResults(ctx context.Context, sessionID string, results []domain.ClaimResult) error
	FinishSession(ctx context.Context, sessionID string) error
}

// MemStore is an in-memory Store. IDs are sess-N and taught-N.
type MemStore struct {
	mu       sync.Mutex
	concepts map[string]domain.Concept
	sources  []domain.Source
	n        int

	Sessions  map[string]domain.TeachSession
	Taught    map[string][]domain.TaughtClaim
	Assistant map[string]map[int]string
	Results   map[string][]domain.ClaimResult
	Finished  map[string]bool
}

func NewMemStore(sources []domain.Source, concepts ...domain.Concept) *MemStore {
	m := &MemStore{
		concepts: map[string]domain.Concept{}, sources: sources,
		Sessions: map[string]domain.TeachSession{}, Taught: map[string][]domain.TaughtClaim{},
		Assistant: map[string]map[int]string{}, Results: map[string][]domain.ClaimResult{}, Finished: map[string]bool{},
	}
	for _, c := range concepts {
		m.concepts[c.ID] = c
	}
	return m
}

func (m *MemStore) LoadConcept(_ context.Context, id string) (domain.Concept, []domain.Source, error) {
	c, ok := m.concepts[id]
	if !ok {
		return domain.Concept{}, nil, fmt.Errorf("teach: unknown concept %q", id)
	}
	return c, m.sources, nil
}

func (m *MemStore) StartSession(_ context.Context, conceptID string) (domain.TeachSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.n++
	s := domain.TeachSession{ID: fmt.Sprintf("sess-%d", m.n), ConceptID: conceptID}
	m.Sessions[s.ID] = s
	return s, nil
}

func (m *MemStore) SaveTurn(_ context.Context, t domain.TaughtClaim, _, _ string) (domain.TaughtClaim, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.n++
	t.ID = fmt.Sprintf("taught-%d", m.n)
	m.Taught[t.SessionID] = append(m.Taught[t.SessionID], t)
	return t, nil
}

func (m *MemStore) SetAssistantText(_ context.Context, sid string, turnNo int, text string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Assistant[sid] == nil {
		m.Assistant[sid] = map[int]string{}
	}
	m.Assistant[sid][turnNo] = text
	return nil
}

func (m *MemStore) SaveResults(_ context.Context, sid string, r []domain.ClaimResult) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Results[sid] = r
	return nil
}

func (m *MemStore) FinishSession(_ context.Context, sid string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Finished[sid] = true
	return nil
}
