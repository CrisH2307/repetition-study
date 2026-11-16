package app

import (
	"context"
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
	// Packs loaded by cmd/seed leave sources.subject_id empty, so also take the
	// sources the course's claims cite (the Phase 0 rule). Ownership is checked above.
	_, cited, err := u.Store.LoadConcept(ctx, id)
	if err != nil {
		return domain.Concept{}, nil, err
	}
	seen := map[string]bool{}
	all := srcs[:0:0]
	for _, x := range append(srcs, cited...) {
		if !seen[x.ID] {
			seen[x.ID] = true
			all = append(all, x)
		}
	}
	return c, all, nil
}

// capLLM refuses calls once the app has spent cap dollars in the calendar
// month (UTC), so checks report unchecked and no date moves (D33).
// ponytail: the spend is counted in memory, so a restart starts the month at zero.
// Persist it (a table) if the cap must survive restarts.
type capLLM struct {
	domain.LLM
	cap   float64
	total func() float64 // all spend so far, ever
	now   func() time.Time

	mu    sync.Mutex
	month time.Month
	year  int
	base  float64 // total at the start of this month
}

func (c *capLLM) spent() float64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	t, now := c.total(), c.now().UTC()
	if now.Year() != c.year || now.Month() != c.month {
		c.year, c.month, c.base = now.Year(), now.Month(), t
	}
	return t - c.base
}

func (c *capLLM) JSON(ctx context.Context, req domain.LLMRequest, out any) error {
	if c.spent() >= c.cap {
		return llm.ErrBudget
	}
	return c.LLM.JSON(ctx, req, out)
}

func (c *capLLM) Stream(ctx context.Context, req domain.LLMRequest) (<-chan domain.Delta, error) {
	if c.spent() >= c.cap {
		return nil, llm.ErrBudget
	}
	return c.LLM.Stream(ctx, req)
}
