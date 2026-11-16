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
	return c, srcs, nil
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
