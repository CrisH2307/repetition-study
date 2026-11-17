// Package app is the composition root: it wires the real implementations
// together and serves api/openapi.yaml over net/http. With Deps.Auth set every
// route but /healthz and /v1/auth/redeem needs a sign-in and acts as that user.
// Without it (Phase 0 tests) every request acts as the seeded demo user.
package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/CrisH2307/repetition-study/internal/auth"
	"github.com/CrisH2307/repetition-study/internal/domain"
)

// Deps is everything the handlers talk to. Reports comes from Teach when the
// engine also implements domain.ReportReader.
type Deps struct {
	Teach    domain.TeachEngine
	Sched    domain.Scheduler
	Due      func(ctx context.Context, now time.Time) ([]domain.DueItem, error) // optional, defaults to Sched.Due
	Planner  domain.Planner
	Subjects SubjectStore
	Clock    domain.Clock
	Log      *slog.Logger

	// Contract v3. Auth nil means Phase 0 mode: no sign-in, no v3 routes.
	Auth         domain.Auth
	SecureCookie bool // the session cookie needs HTTPS; false only for http://localhost
	Course       domain.CourseStore
	Reviews      domain.ReviewStore
	Notes        domain.CourseNotes
	Today        domain.TodayPlanner
	Own          Owner
	// PlannerFor returns the planner acting for one user. Optional, defaults to Planner.
	PlannerFor func(userID string) domain.Planner

	// Contract v4. Needed whenever Auth is set.
	Feedback domain.FeedbackStore
	Privacy  domain.PrivacyStore
	Imports  domain.ImportStore
	Importer domain.Importer
}

// Owner answers "is this Phase 0 id the user's?" with domain.ErrNotFound for no.
type Owner interface {
	Concept(ctx context.Context, userID, conceptID string) error
	Card(ctx context.Context, userID, cardID string) error
	// ConceptIDs is every concept of the user's live courses.
	ConceptIDs(ctx context.Context, userID string) (map[string]bool, error)
}

// SubjectStore is the little persistence the subjects endpoints need.
type SubjectStore interface {
	List(ctx context.Context, userID string) ([]domain.Subject, error)
	Create(ctx context.Context, userID string, s domain.Subject) (domain.Subject, error)
}

type api struct {
	Deps
	reports domain.ReportReader
}

// uid is the user a request acts as.
func (a *api) uid(r *http.Request) string {
	if a.Auth == nil {
		return domain.DemoUserID
	}
	u, _ := auth.UserFrom(r.Context())
	return u.ID
}

// ctx carries the user to code that has no user argument (the teach store).
func (a *api) ctx(r *http.Request) context.Context {
	return context.WithValue(r.Context(), userKey{}, a.uid(r))
}

type userKey struct{}

// ownsSession answers 404 for another user's teach session. Phase 0 mode skips it.
func (a *api) ownsSession(w http.ResponseWriter, r *http.Request) bool {
	if a.Auth == nil {
		return true
	}
	if err := a.Reviews.SessionOwner(r.Context(), a.uid(r), r.PathValue("id")); err != nil {
		a.fail(w, r, err)
		return false
	}
	return true
}

// owns runs one Owner check. Phase 0 mode skips it.
func (a *api) owns(w http.ResponseWriter, r *http.Request, check func(ctx context.Context, userID, id string) error) bool {
	if a.Auth == nil {
		return true
	}
	if err := check(r.Context(), a.uid(r), r.PathValue("id")); err != nil {
		a.fail(w, r, err)
		return false
	}
	return true
}

const maxBody = 1 << 20 // 1 MiB: turns are text and LaTeX only

// NewHandler returns the router. Routes not listed here (digest, plan read and
// edit, card edit) are deliberately absent in Phase 0 and answer 404.
func NewHandler(d Deps) http.Handler {
	if d.Log == nil {
		d.Log = slog.Default()
	}
	a := &api{Deps: d}
	rr, ok := d.Teach.(domain.ReportReader)
	if !ok {
		panic("app: the teach engine must implement domain.ReportReader")
	}
	a.reports = rr
	mux := http.NewServeMux()
	health := func(w http.ResponseWriter, _ *http.Request) { writeJSON(w, 200, map[string]string{"status": "ok"}) }
	mux.HandleFunc("GET /v1/subjects", a.listSubjects)
	mux.HandleFunc("POST /v1/subjects", a.createSubject)
	mux.HandleFunc("POST /v1/plan/generate", a.generatePlan)
	mux.HandleFunc("GET /v1/analytics/further-planning", a.furtherPlanning)
	mux.HandleFunc("POST /v1/teach/sessions", a.startSession)
	mux.HandleFunc("POST /v1/teach/sessions/{id}/turns", a.turn)
	mux.HandleFunc("POST /v1/teach/sessions/{id}/finish", a.finish)
	mux.HandleFunc("GET /v1/teach/sessions/{id}/report", a.report)
	mux.HandleFunc("GET /v1/reviews/due", a.due)
	mux.HandleFunc("POST /v1/cards/{id}/review", a.reviewCard)
	mux.HandleFunc("POST /v1/concepts/{id}/override", a.override)
	if d.Auth == nil {
		mux.HandleFunc("GET /healthz", health)
		return mux
	}
	a.routesV3(mux)
	a.routesV4(mux)
	ah := auth.HTTP{Auth: d.Auth, Secure: d.SecureCookie}
	top := http.NewServeMux()
	top.HandleFunc("GET /healthz", health)
	top.HandleFunc("POST /v1/auth/redeem", ah.Redeem)
	mux.HandleFunc("POST /v1/auth/sign-out", ah.SignOut)
	top.Handle("/", ah.Require(mux))
	return auth.Protect(top)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

// fail maps an error to a status. Unknown ids are 404. Anything else that the
// caller did not cause is a 500 whose detail stays in the log.
func (a *api) fail(w http.ResponseWriter, r *http.Request, err error) {
	code, msg := 500, "internal error"
	var ue *domain.UserError
	switch {
	case errors.Is(err, context.Canceled):
		// The caller hung up (Finish returns ctx.Err()). Nobody is listening, so log quietly.
		a.Log.Info("request canceled by caller", "method", r.Method, "path", r.URL.Path)
		writeJSON(w, 499, map[string]string{"error": "request canceled"})
		return
	case errors.Is(err, context.DeadlineExceeded):
		// The caller's deadline passed mid-request (Finish returns ctx.Err()). Not a server fault.
		a.Log.Info("request deadline passed", "method", r.Method, "path", r.URL.Path)
		code, msg = 504, "timed out"
	case isNotFound(err):
		code, msg = 404, "not found"
	case errors.As(err, &ue):
		code, msg = 400, ue.Msg
	case strings.Contains(err.Error(), "already finished"):
		code, msg = 409, "session already finished"
	}
	if code == 500 {
		a.Log.Error("request failed", "method", r.Method, "path", r.URL.Path, "err", err)
	}
	writeJSON(w, code, map[string]string{"error": msg})
}

func bad(w http.ResponseWriter, msg string) { writeJSON(w, 400, map[string]string{"error": msg}) }

// isNotFound covers the three ways a missing id shows up: no row, an id that is
// not a uuid (22P02), and a foreign key to nothing (23503). The teach engine
// has no sentinel error, so its "unknown session" text is matched too.
// ponytail: string match until internal/teach exports an error value.
func isNotFound(err error) bool {
	if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, domain.ErrNotFound) {
		return true
	}
	var pe *pgconn.PgError
	if errors.As(err, &pe) && (pe.Code == "22P02" || pe.Code == "23503") {
		return true
	}
	s := err.Error()
	return strings.Contains(s, "unknown session") || strings.Contains(s, "no report")
}

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBody))
	if err := dec.Decode(v); err != nil {
		bad(w, "invalid JSON body")
		return false
	}
	return true
}

func parseDate(s string) (time.Time, error) { return time.Parse("2006-01-02", s) }

// ---- subjects ----

type subjectOut struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	WeeklyHours float64 `json:"weekly_hours"`
	Level       string  `json:"level"`
	ExamDate    *string `json:"exam_date,omitempty"` // the contract says format date, not date-time
	Weight      float64 `json:"weight"`
}

func toSubjectOut(s domain.Subject) subjectOut {
	o := subjectOut{ID: s.ID, Name: s.Name, WeeklyHours: s.WeeklyHours, Level: string(s.Level), Weight: s.Weight}
	if s.ExamDate != nil {
		d := s.ExamDate.UTC().Format("2006-01-02")
		o.ExamDate = &d
	}
	return o
}

func (a *api) listSubjects(w http.ResponseWriter, r *http.Request) {
	subs, err := a.Subjects.List(r.Context(), a.uid(r))
	if err != nil {
		a.fail(w, r, err)
		return
	}
	out := make([]subjectOut, 0, len(subs))
	for _, s := range subs {
		out = append(out, toSubjectOut(s))
	}
	writeJSON(w, 200, out)
}

// readSubject decodes and checks the body of POST /v1/subjects, POST /v1/courses and PATCH /v1/courses/{id}.
func readSubject(w http.ResponseWriter, r *http.Request) (domain.Subject, bool) {
	var in struct {
		Name        string   `json:"name"`
		WeeklyHours *float64 `json:"weekly_hours"`
		Level       string   `json:"level"`
		ExamDate    *string  `json:"exam_date"`
		Weight      *float64 `json:"weight"`
	}
	if !decode(w, r, &in) {
		return domain.Subject{}, false
	}
	s := domain.Subject{Name: strings.TrimSpace(in.Name), Level: domain.Level(in.Level), Weight: 1}
	switch {
	case s.Name == "":
		bad(w, "name is required")
		return s, false
	case in.WeeklyHours == nil || *in.WeeklyHours < 0:
		bad(w, "weekly_hours must be 0 or more")
		return s, false
	case s.Level != domain.LevelBeginner && s.Level != domain.LevelIntermediate && s.Level != domain.LevelAdvanced:
		bad(w, "level must be beginner, intermediate or advanced")
		return s, false
	case in.Weight != nil && *in.Weight < 0:
		bad(w, "weight must be 0 or more")
		return s, false
	}
	s.WeeklyHours = *in.WeeklyHours
	if in.Weight != nil {
		s.Weight = *in.Weight
	}
	if in.ExamDate != nil {
		d, err := parseDate(*in.ExamDate)
		if err != nil {
			bad(w, "exam_date must be YYYY-MM-DD")
			return s, false
		}
		s.ExamDate = &d
	}
	return s, true
}

func (a *api) createSubject(w http.ResponseWriter, r *http.Request) {
	s, ok := readSubject(w, r)
	if !ok {
		return
	}
	got, err := a.Subjects.Create(r.Context(), a.uid(r), s)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	writeJSON(w, 201, toSubjectOut(got))
}

// ---- plan ----

func (a *api) plannerFor(r *http.Request) domain.Planner {
	if a.PlannerFor != nil {
		return a.PlannerFor(a.uid(r))
	}
	return a.Planner
}

func (a *api) generatePlan(w http.ResponseWriter, r *http.Request) {
	var in struct {
		WeekStart string `json:"week_start"`
	}
	if !decode(w, r, &in) {
		return
	}
	ws, err := parseDate(in.WeekStart)
	if err != nil {
		bad(w, "week_start must be YYYY-MM-DD")
		return
	}
	p, err := a.plannerFor(r).Generate(r.Context(), a.uid(r), ws)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	if p.Blocks == nil {
		p.Blocks = []domain.PlanBlock{}
	}
	writeJSON(w, 200, p)
}

func (a *api) furtherPlanning(w http.ResponseWriter, r *http.Request) {
	ws, err := parseDate(r.URL.Query().Get("week"))
	if err != nil {
		bad(w, "week must be YYYY-MM-DD")
		return
	}
	fp, err := a.plannerFor(r).FurtherPlanning(r.Context(), ws)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	if fp.Subjects == nil {
		fp.Subjects = []domain.SubjectProgress{}
	}
	for i := range fp.Subjects {
		if fp.Subjects[i].Concepts == nil {
			fp.Subjects[i].Concepts = []domain.ConceptProgress{}
		}
	}
	writeJSON(w, 200, fp)
}

// ---- teach ----

func (a *api) startSession(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ConceptID string `json:"concept_id"`
	}
	if !decode(w, r, &in) {
		return
	}
	if in.ConceptID == "" {
		bad(w, "concept_id is required")
		return
	}
	if a.Auth != nil {
		if err := a.Own.Concept(r.Context(), a.uid(r), in.ConceptID); err != nil {
			a.fail(w, r, err)
			return
		}
	}
	ts, err := a.Teach.Start(a.ctx(r), in.ConceptID)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	writeJSON(w, 201, ts)
}

// turn streams the AI student's question as server-sent events: token, then
// done, or error. Transcript and latex go to the engine as plain data.
func (a *api) turn(w http.ResponseWriter, r *http.Request) {
	if !a.ownsSession(w, r) {
		return
	}
	var in struct {
		Transcript string `json:"transcript"`
		LaTeX      string `json:"latex"`
	}
	if !decode(w, r, &in) {
		return
	}
	if strings.TrimSpace(in.Transcript) == "" && strings.TrimSpace(in.LaTeX) == "" {
		bad(w, "transcript is required")
		return
	}
	ch, err := a.Teach.Turn(r.Context(), r.PathValue("id"), in.Transcript, in.LaTeX)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	rc := http.NewResponseController(w)
	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	h.Set("X-Accel-Buffering", "no")
	w.WriteHeader(200)
	send := func(event string, data any) {
		b, _ := json.Marshal(data)
		rc.SetWriteDeadline(time.Now().Add(30 * time.Second)) // the server write timeout applies per write, not per stream
		fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, b)
		rc.Flush()
	}
	for d := range ch {
		if d.Err != nil {
			a.Log.Warn("turn stream failed", "session", r.PathValue("id"), "err", d.Err)
			send("error", map[string]string{"error": "the student could not answer"})
			for range ch { // let the producer finish
			}
			return
		}
		send("token", map[string]string{"text": d.Text})
	}
	send("done", struct{}{})
}

func (a *api) finish(w http.ResponseWriter, r *http.Request) {
	if !a.ownsSession(w, r) {
		return
	}
	rep, err := a.Teach.Finish(r.Context(), r.PathValue("id"))
	if err != nil {
		a.fail(w, r, err)
		return
	}
	writeJSON(w, 200, normalize(rep))
}

func (a *api) report(w http.ResponseWriter, r *http.Request) {
	if !a.ownsSession(w, r) {
		return
	}
	rep, err := a.reports.GetReport(r.Context(), r.PathValue("id"))
	if err != nil {
		a.fail(w, r, err)
		return
	}
	writeJSON(w, 200, normalize(rep))
}

// normalize makes sure the results array is never null.
func normalize(rep domain.Report) domain.Report {
	if rep.Results == nil {
		rep.Results = []domain.ClaimResult{}
	}
	return rep
}

// ---- reviews ----

func (a *api) due(w http.ResponseWriter, r *http.Request) {
	f := a.Due
	if f == nil {
		f = a.Sched.Due
	}
	items, err := f(r.Context(), a.Clock.Now().UTC())
	if err != nil {
		a.fail(w, r, err)
		return
	}
	if a.Auth != nil { // the scheduler is not user-aware: keep only this user's concepts
		mine, err := a.Own.ConceptIDs(r.Context(), a.uid(r))
		if err != nil {
			a.fail(w, r, err)
			return
		}
		kept := items[:0:0]
		for _, it := range items {
			if mine[it.ConceptID] {
				kept = append(kept, it)
			}
		}
		items = kept
	}
	if items == nil {
		items = []domain.DueItem{}
	}
	writeJSON(w, 200, items)
}

func (a *api) reviewCard(w http.ResponseWriter, r *http.Request) {
	if a.Auth != nil && !a.owns(w, r, a.Own.Card) {
		return
	}
	var in struct {
		Grade domain.Grade `json:"grade"`
	}
	if !decode(w, r, &in) {
		return
	}
	switch in.Grade {
	case domain.GradeAgain, domain.GradeHard, domain.GradeGood, domain.GradeEasy:
	default:
		bad(w, "grade must be again, hard, good or easy")
		return
	}
	ev, err := a.Sched.ReviewCard(r.Context(), r.PathValue("id"), in.Grade)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	writeJSON(w, 200, ev)
}

func (a *api) override(w http.ResponseWriter, r *http.Request) {
	if a.Auth != nil && !a.owns(w, r, a.Own.Concept) {
		return
	}
	var in struct {
		DueAt  time.Time `json:"due_at"`
		Reason string    `json:"reason"`
	}
	if !decode(w, r, &in) {
		return
	}
	if in.DueAt.IsZero() || strings.TrimSpace(in.Reason) == "" {
		bad(w, "due_at (RFC 3339) and reason are required")
		return
	}
	ev, err := a.Sched.Override(r.Context(), r.PathValue("id"), in.DueAt.UTC(), strings.TrimSpace(in.Reason))
	if err != nil {
		a.fail(w, r, err)
		return
	}
	writeJSON(w, 200, ev)
}

func authUser(r *http.Request) (domain.User, bool) { return auth.UserFrom(r.Context()) }
