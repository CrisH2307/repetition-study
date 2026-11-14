// Package app is the composition root: it wires the real implementations
// together and serves api/openapi.yaml over net/http. Every request acts as the
// seeded demo user (no auth in Phase 0).
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
}

// SubjectStore is the little persistence the subjects endpoints need.
type SubjectStore interface {
	List(ctx context.Context) ([]domain.Subject, error)
	Create(ctx context.Context, s domain.Subject) (domain.Subject, error)
}

type api struct {
	Deps
	reports domain.ReportReader
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
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { writeJSON(w, 200, map[string]string{"status": "ok"}) })
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
	return mux
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
	if errors.Is(err, pgx.ErrNoRows) {
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
	subs, err := a.Subjects.List(r.Context())
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

func (a *api) createSubject(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name        string   `json:"name"`
		WeeklyHours *float64 `json:"weekly_hours"`
		Level       string   `json:"level"`
		ExamDate    *string  `json:"exam_date"`
		Weight      *float64 `json:"weight"`
	}
	if !decode(w, r, &in) {
		return
	}
	s := domain.Subject{Name: strings.TrimSpace(in.Name), Level: domain.Level(in.Level), Weight: 1}
	switch {
	case s.Name == "":
		bad(w, "name is required")
		return
	case in.WeeklyHours == nil || *in.WeeklyHours < 0:
		bad(w, "weekly_hours must be 0 or more")
		return
	case s.Level != domain.LevelBeginner && s.Level != domain.LevelIntermediate && s.Level != domain.LevelAdvanced:
		bad(w, "level must be beginner, intermediate or advanced")
		return
	case in.Weight != nil && *in.Weight < 0:
		bad(w, "weight must be 0 or more")
		return
	}
	s.WeeklyHours = *in.WeeklyHours
	if in.Weight != nil {
		s.Weight = *in.Weight
	}
	if in.ExamDate != nil {
		d, err := parseDate(*in.ExamDate)
		if err != nil {
			bad(w, "exam_date must be YYYY-MM-DD")
			return
		}
		s.ExamDate = &d
	}
	got, err := a.Subjects.Create(r.Context(), s)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	writeJSON(w, 201, toSubjectOut(got))
}

// ---- plan ----

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
	p, err := a.Planner.Generate(r.Context(), domain.DemoUserID, ws)
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
	fp, err := a.Planner.FurtherPlanning(r.Context(), ws)
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
	ts, err := a.Teach.Start(r.Context(), in.ConceptID)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	writeJSON(w, 201, ts)
}

// turn streams the AI student's question as server-sent events: token, then
// done, or error. Transcript and latex go to the engine as plain data.
func (a *api) turn(w http.ResponseWriter, r *http.Request) {
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
	rep, err := a.Teach.Finish(r.Context(), r.PathValue("id"))
	if err != nil {
		a.fail(w, r, err)
		return
	}
	writeJSON(w, 200, normalize(rep))
}

func (a *api) report(w http.ResponseWriter, r *http.Request) {
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
	if items == nil {
		items = []domain.DueItem{}
	}
	writeJSON(w, 200, items)
}

func (a *api) reviewCard(w http.ResponseWriter, r *http.Request) {
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
