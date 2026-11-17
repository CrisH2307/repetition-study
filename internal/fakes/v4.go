package fakes

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"sync"
	"time"

	"github.com/CrisH2307/repetition-study/internal/domain"
)

// Contract v4 fakes. In memory, scoped to the user id they are given.

var (
	_ domain.FeedbackStore = (*FeedbackStore)(nil)
	_ domain.PrivacyStore  = (*PrivacyStore)(nil)
	_ domain.ImportStore   = (*ImportStore)(nil)
	_ domain.Importer      = (*Importer)(nil)
	_ domain.EvalCaseStore = (*EvalCaseStore)(nil)
	_ domain.MetricsStore  = (*MetricsStore)(nil)
	_ domain.SpendLedger   = (*SpendLedger)(nil)
)

type FeedbackStore struct {
	mu       sync.Mutex
	answers  map[string]domain.FeedbackAnswer // user|week
	consents map[string][]domain.Consent
}

func weekKey(user string, w time.Time) string { return user + "|" + w.Format(time.DateOnly) }

func (f *FeedbackStore) SaveFeedback(_ context.Context, user string, a domain.FeedbackAnswer) (domain.FeedbackAnswer, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.answers == nil {
		f.answers = map[string]domain.FeedbackAnswer{}
	}
	a.CreatedAt = Epoch
	f.answers[weekKey(user, a.WeekStart)] = a
	return a, nil
}

func (f *FeedbackStore) GetFeedback(_ context.Context, user string, week time.Time) (domain.FeedbackAnswer, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	a, ok := f.answers[weekKey(user, week)]
	if !ok {
		return domain.FeedbackAnswer{}, domain.ErrNotFound
	}
	return a, nil
}

func (f *FeedbackStore) SaveConsent(_ context.Context, user, version string, at time.Time) (domain.Consent, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.consents == nil {
		f.consents = map[string][]domain.Consent{}
	}
	for _, c := range f.consents[user] {
		if c.TextVersion == version {
			return c, nil
		}
	}
	c := domain.Consent{TextVersion: version, AcceptedAt: at}
	f.consents[user] = append(f.consents[user], c)
	return c, nil
}

func (f *FeedbackStore) Consents(_ context.Context, user string) ([]domain.Consent, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]domain.Consent{}, f.consents[user]...), nil
}

// PrivacyStore returns what tests put in Exports. DeleteUser removes the
// user's export and records the id in Deleted.
type PrivacyStore struct {
	mu      sync.Mutex
	Exports map[string]domain.UserExport
	Deleted []string
}

func (p *PrivacyStore) ExportUser(_ context.Context, user string, now time.Time) (domain.UserExport, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	e, ok := p.Exports[user]
	if !ok {
		return domain.UserExport{}, domain.ErrNotFound
	}
	e.ExportedAt = now
	return e, nil
}

func (p *PrivacyStore) DeleteUser(_ context.Context, user string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.Exports[user]; !ok {
		return domain.ErrNotFound
	}
	delete(p.Exports, user)
	p.Deleted = append(p.Deleted, user)
	return nil
}

// ImportStore keeps jobs; Owners maps subject id to user id and must be set
// for CreateImport to accept a course.
type ImportStore struct {
	Owners map[string]string

	mu   sync.Mutex
	ids  ids
	jobs map[string]domain.ImportJob
}

func (s *ImportStore) owned(user string, j domain.ImportJob, ok bool) (domain.ImportJob, error) {
	if !ok || s.Owners[j.SubjectID] != user {
		return domain.ImportJob{}, domain.ErrNotFound
	}
	return j, nil
}

func (s *ImportStore) CreateImport(_ context.Context, user, subjectID, filename string) (domain.ImportJob, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Owners[subjectID] != user {
		return domain.ImportJob{}, domain.ErrNotFound
	}
	if s.jobs == nil {
		s.jobs = map[string]domain.ImportJob{}
	}
	j := domain.ImportJob{ID: s.ids.next("import"), SubjectID: subjectID, Filename: filename,
		Status: domain.ImportRunning, RowErrors: []domain.ImportRowError{}, CreatedAt: Epoch}
	s.jobs[j.ID] = j
	return j, nil
}

func (s *ImportStore) GetImport(_ context.Context, user, id string) (domain.ImportJob, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	j, ok := s.jobs[id]
	return s.owned(user, j, ok)
}

func (s *ImportStore) FinishImport(_ context.Context, user string, j domain.ImportJob, at time.Time) (domain.ImportJob, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	old, ok := s.jobs[j.ID]
	if _, err := s.owned(user, old, ok); err != nil {
		return domain.ImportJob{}, err
	}
	old.Status, old.RowsTotal, old.RowsImported, old.RowErrors, old.FinishedAt = j.Status, j.RowsTotal, j.RowsImported, j.RowErrors, &at
	s.jobs[j.ID] = old
	return old, nil
}

func (s *ImportStore) ListImports(_ context.Context, user, subjectID string) ([]domain.ImportJob, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Owners[subjectID] != user {
		return nil, domain.ErrNotFound
	}
	out := []domain.ImportJob{}
	for i := 1; i <= s.ids.n["import"]; i++ {
		if j := s.jobs[fmt.Sprintf("import-%d", i)]; j.SubjectID == subjectID {
			out = append(out, j)
		}
	}
	return out, nil
}

// Importer finishes at once. It counts data rows and refuses rows without four
// non-empty columns; it does not create claims (the real one uses CourseNotes).
type Importer struct{ Store *ImportStore }

func (im *Importer) Import(ctx context.Context, user, subjectID, filename string, data []byte) (domain.ImportJob, error) {
	j, err := im.Store.CreateImport(ctx, user, subjectID, filename)
	if err != nil {
		return j, err
	}
	r := csv.NewReader(bytes.NewReader(data))
	r.FieldsPerRecord = -1
	rows, err := r.ReadAll()
	if err != nil || len(rows) == 0 {
		j.Status = domain.ImportFailed
		return im.Store.FinishImport(ctx, user, j, Epoch)
	}
	j.Status = domain.ImportDone
	for i, row := range rows[1:] {
		j.RowsTotal++
		if len(row) != 4 || row[0] == "" || row[1] == "" || row[2] == "" || row[3] == "" {
			j.RowErrors = append(j.RowErrors, domain.ImportRowError{Row: i + 2, Error: "This row needs concept, claim, kind and quote."})
			continue
		}
		j.RowsImported++
	}
	return im.Store.FinishImport(ctx, user, j, Epoch)
}

// EvalCaseStore keeps disputes as cases. Disputes holds the candidates by id;
// ResolveDispute moves upheld ones into the result.
type EvalCaseStore struct {
	mu       sync.Mutex
	Disputes map[string]domain.DisputeCase // ExpectedLabel is filled on uphold
	order    []string
	resolved map[string]domain.DisputeOutcome
}

func (e *EvalCaseStore) ResolveDispute(_ context.Context, id string, o domain.DisputeOutcome, expected *domain.ClaimLabel) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	c, ok := e.Disputes[id]
	if !ok {
		return domain.ErrNotFound
	}
	if (o == domain.DisputeUpheld) != (expected != nil) || (o != domain.DisputeUpheld && o != domain.DisputeRejected) {
		return &domain.UserError{Msg: "An upheld dispute needs the right label; a rejected one has none."}
	}
	if e.resolved == nil {
		e.resolved = map[string]domain.DisputeOutcome{}
	}
	if _, seen := e.resolved[id]; !seen {
		e.order = append(e.order, id)
	}
	e.resolved[id] = o
	if expected != nil {
		c.ExpectedLabel = *expected
		e.Disputes[id] = c
	}
	return nil
}

func (e *EvalCaseStore) UpheldDisputes(context.Context) ([]domain.DisputeCase, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	out := []domain.DisputeCase{}
	for _, id := range e.order {
		if e.resolved[id] == domain.DisputeUpheld {
			out = append(out, e.Disputes[id])
		}
	}
	return out, nil
}

// MetricsStore returns what tests put in it.
type MetricsStore struct {
	Weeks []domain.WeekCounts
	Users []domain.UserWeeks
}

func (m *MetricsStore) WeekCounts(_ context.Context, from, to time.Time) ([]domain.WeekCounts, error) {
	out := []domain.WeekCounts{}
	for _, w := range m.Weeks {
		if !w.WeekStart.Before(from) && w.WeekStart.Before(to) {
			out = append(out, w)
		}
	}
	return out, nil
}

func (m *MetricsStore) UserWeeks(context.Context) ([]domain.UserWeeks, error) {
	return append([]domain.UserWeeks{}, m.Users...), nil
}

// SpendLedger sums USD per UTC month.
type SpendLedger struct {
	mu     sync.Mutex
	months map[string]float64
}

func (s *SpendLedger) Add(_ context.Context, at time.Time, usd float64) (float64, error) {
	if usd < 0 {
		return 0, fmt.Errorf("spend must not be negative: %v", usd)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.months == nil {
		s.months = map[string]float64{}
	}
	k := at.UTC().Format("2006-01")
	s.months[k] += usd
	return s.months[k], nil
}

func (s *SpendLedger) MonthTotal(_ context.Context, at time.Time) (float64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.months[at.UTC().Format("2006-01")], nil
}
