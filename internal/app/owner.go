package app

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/CrisH2307/repetition-study/internal/domain"
	"github.com/CrisH2307/repetition-study/internal/llm"
	"github.com/CrisH2307/repetition-study/internal/store"
)

// pgOwner answers ownership for the Phase 0 ids (concepts and cards).
type pgOwner struct {
	db     DB
	course domain.CourseStore
}

func (o pgOwner) Concept(ctx context.Context, userID, id string) error {
	_, err := o.course.GetConcept(ctx, userID, id)
	return err
}

func (o pgOwner) Card(ctx context.Context, userID, id string) error {
	var ok bool
	err := o.db.QueryRow(ctx, `SELECT true FROM cards ca JOIN concepts k ON k.id = ca.concept_id
		JOIN subjects s ON s.id = k.subject_id
		WHERE ca.id = $1 AND s.user_id = $2 AND s.archived_at IS NULL`, id, userID).Scan(&ok)
	if err != nil && isNotFound(err) {
		return domain.ErrNotFound
	}
	return err
}

func (o pgOwner) ConceptIDs(ctx context.Context, userID string) (map[string]bool, error) {
	rows, err := o.db.Query(ctx, `SELECT k.id::text FROM concepts k JOIN subjects s ON s.id = k.subject_id
		WHERE s.user_id = $1 AND s.archived_at IS NULL`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out[id] = true
	}
	return out, rows.Err()
}

// userTeach is the teach engine's store with LoadConcept limited to the
// signed-in user: their live claims and their course's sources, which hold only
// text from pages they marked Ready. The user comes from the request context.
type userTeach struct{ *store.Store }

func (u userTeach) LoadConcept(ctx context.Context, id string) (domain.Concept, []domain.Source, error) {
	user, _ := ctx.Value(userKey{}).(string)
	if user == "" {
		user = domain.DemoUserID // Phase 0 mode
	}
	c, err := u.Store.GetConcept(ctx, user, id)
	if err != nil {
		return domain.Concept{}, nil, err
	}
	req := c.Claims[:0:0]
	for _, cl := range c.Claims {
		if cl.Required {
			req = append(req, cl)
		}
	}
	c.Claims = req
	srcs, err := u.Store.CourseSources(ctx, user, c.SubjectID)
	if err != nil {
		return domain.Concept{}, nil, err
	}
	return c, srcs, nil
}

// spendRec copies model spend from the in-memory meter into the ledger, so
// the monthly cap survives restarts (D33).
type spendRec struct {
	ledger domain.SpendLedger
	total  func() float64 // all metered spend since this process started
	now    func() time.Time

	mu   sync.Mutex
	last float64 // part of total already in the ledger
}

// flush adds spend not yet in the ledger. On a ledger error it keeps the amount for the next flush.
func (s *spendRec) flush(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t := s.total()
	if t <= s.last {
		return
	}
	if _, err := s.ledger.Add(context.WithoutCancel(ctx), s.now().UTC(), t-s.last); err == nil {
		s.last = t
	}
}

// metered records spend after each call of the wrapped model (a stream counts when it ends).
type metered struct {
	domain.LLM
	rec *spendRec
}

func (m metered) JSON(ctx context.Context, req domain.LLMRequest, out any) error {
	defer m.rec.flush(ctx)
	return m.LLM.JSON(ctx, req, out)
}

func (m metered) Stream(ctx context.Context, req domain.LLMRequest) (<-chan domain.Delta, error) {
	in, err := m.LLM.Stream(ctx, req)
	if err != nil {
		m.rec.flush(ctx)
		return nil, err
	}
	out := make(chan domain.Delta)
	go func() {
		defer close(out)
		defer m.rec.flush(ctx)
		for d := range in {
			out <- d
		}
	}()
	return out, nil
}

// capLLM refuses calls once the ledger shows cap dollars spent in the calendar
// month (UTC), so checks report unchecked and no date moves (D33). It fails
// closed: if the ledger cannot be read, the call is refused.
type capLLM struct {
	domain.LLM
	cap    float64
	ledger domain.SpendLedger
	rec    *spendRec // optional: flushes meter spend into the ledger before the check
	now    func() time.Time
}

func (c *capLLM) check(ctx context.Context) error {
	if c.rec != nil {
		c.rec.flush(ctx)
	}
	spent, err := c.ledger.MonthTotal(ctx, c.now().UTC())
	if err != nil {
		return fmt.Errorf("spend ledger: %w", err)
	}
	if spent >= c.cap {
		return llm.ErrBudget
	}
	return nil
}

func (c *capLLM) JSON(ctx context.Context, req domain.LLMRequest, out any) error {
	if err := c.check(ctx); err != nil {
		return err
	}
	return c.LLM.JSON(ctx, req, out)
}

func (c *capLLM) Stream(ctx context.Context, req domain.LLMRequest) (<-chan domain.Delta, error) {
	if err := c.check(ctx); err != nil {
		return nil, err
	}
	return c.LLM.Stream(ctx, req)
}

// failInterrupted runs once at start. A restart kills the background reads, so
// any upload still reading gets a plain error and the student is never stuck.
func failInterrupted(ctx context.Context, db DB, course domain.CourseStore) (int, error) {
	rows, err := db.Query(ctx, `SELECT s.user_id::text, u.id::text FROM uploads u JOIN subjects s ON s.id = u.subject_id WHERE u.status = 'reading'`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	var pairs [][2]string
	for rows.Next() {
		var p [2]string
		if err := rows.Scan(&p[0], &p[1]); err != nil {
			return 0, err
		}
		pairs = append(pairs, p)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}
	for _, p := range pairs {
		if err := course.SetUploadStatus(ctx, p[0], p[1], domain.UploadNeedsChecking, "Reading was interrupted. Please upload the file again."); err != nil {
			return 0, err
		}
	}
	return len(pairs), nil
}
