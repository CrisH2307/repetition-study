package app

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/CrisH2307/repetition-study/internal/auth"
	"github.com/CrisH2307/repetition-study/internal/domain"
)

const (
	maxCSV         = 1 << 20 // a CSV of 500 rows is far smaller than this
	maxAnswer      = 4000
	deleteConfirm  = "delete my account"
	maxTextVersion = 64
)

// weeklyQuestions rotate by week, chosen by code (D40). The first is the G2 question.
var weeklyQuestions = []string{
	"How would you feel if you could no longer use this?",
	"What is the one thing that would make this more useful for you?",
	"What got in your way this week?",
	"Did the app find something you had wrong? How did that feel?",
}

func (a *api) routesV4(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/me/export", a.exportMe)
	mux.HandleFunc("DELETE /v1/me", a.deleteMe)
	mux.HandleFunc("POST /v1/consent", a.consent)
	mux.HandleFunc("GET /v1/feedback", a.getFeedback)
	mux.HandleFunc("POST /v1/feedback", a.postFeedback)
	mux.HandleFunc("POST /v1/courses/{id}/import", a.importCSV)
	mux.HandleFunc("GET /v1/imports/{id}", a.getImport)
}

// weekFor is the user's local Monday as a calendar date at 00:00 UTC, and that week's question.
func weekFor(u domain.User, now time.Time) (time.Time, string) {
	loc := time.UTC
	if l, err := time.LoadLocation(u.TimeZone); err == nil && u.TimeZone != "" {
		loc = l
	}
	y, m, d := now.In(loc).Date()
	day := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	monday := day.AddDate(0, 0, -((int(day.Weekday()) + 6) % 7))
	n := int(monday.Unix() / (7 * 24 * 3600))
	return monday, weeklyQuestions[n%len(weeklyQuestions)]
}

func (a *api) getFeedback(w http.ResponseWriter, r *http.Request) {
	u, _ := authUser(r)
	week, q := weekFor(u, a.Clock.Now())
	p := domain.FeedbackPrompt{WeekStart: week, Question: q}
	got, err := a.Feedback.GetFeedback(r.Context(), u.ID, week)
	switch {
	case err == nil:
		p.Question, p.Answer = got.Question, &got.Answer
	case !errors.Is(err, domain.ErrNotFound):
		a.fail(w, r, err)
		return
	}
	writeJSON(w, 200, p)
}

func (a *api) postFeedback(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Answer string `json:"answer"`
	}
	if !decode(w, r, &in) {
		return
	}
	in.Answer = strings.TrimSpace(in.Answer)
	if in.Answer == "" || len([]rune(in.Answer)) > maxAnswer {
		bad(w, "Write an answer of up to 4000 characters.")
		return
	}
	u, _ := authUser(r)
	week, q := weekFor(u, a.Clock.Now())
	got, err := a.Feedback.SaveFeedback(r.Context(), u.ID, domain.FeedbackAnswer{WeekStart: week, Question: q, Answer: in.Answer})
	if err != nil {
		a.fail(w, r, err)
		return
	}
	writeJSON(w, 201, got)
}

func (a *api) consent(w http.ResponseWriter, r *http.Request) {
	var in struct {
		TextVersion string `json:"text_version"`
	}
	if !decode(w, r, &in) {
		return
	}
	in.TextVersion = strings.TrimSpace(in.TextVersion)
	if in.TextVersion == "" || len(in.TextVersion) > maxTextVersion {
		bad(w, "text_version is required.")
		return
	}
	c, err := a.Feedback.SaveConsent(r.Context(), a.uid(r), in.TextVersion, a.Clock.Now().UTC())
	if err != nil {
		a.fail(w, r, err)
		return
	}
	writeJSON(w, 201, c)
}

func (a *api) exportMe(w http.ResponseWriter, r *http.Request) {
	e, err := a.Privacy.ExportUser(r.Context(), a.uid(r), a.Clock.Now().UTC())
	if err != nil {
		a.fail(w, r, err)
		return
	}
	e.Consents, e.Feedback, e.Courses, e.Concepts = nz(e.Consents), nz(e.Feedback), nz(e.Courses), nz(e.Concepts)
	e.Uploads, e.Pages, e.Sessions, e.Turns = nz(e.Uploads), nz(e.Pages), nz(e.Sessions), nz(e.Turns)
	e.TaughtClaims, e.Reports, e.SelfGrades, e.Disputes = nz(e.TaughtClaims), nz(e.Reports), nz(e.SelfGrades), nz(e.Disputes)
	e.ReviewEvents, e.Imports = nz(e.ReviewEvents), nz(e.Imports)
	w.Header().Set("Content-Disposition", `attachment; filename="my-data.json"`)
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, 200, e)
}

// deleteMe removes the account and everything in it (D43), then clears the cookie.
func (a *api) deleteMe(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Confirm string `json:"confirm"`
	}
	if !decode(w, r, &in) {
		return
	}
	if in.Confirm != deleteConfirm {
		bad(w, `To delete your account, send "delete my account" as the confirmation.`)
		return
	}
	if err := a.Privacy.DeleteUser(r.Context(), a.uid(r)); err != nil {
		a.fail(w, r, err)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name: auth.CookieName, Value: "", Path: "/", MaxAge: -1,
		HttpOnly: true, Secure: a.SecureCookie, SameSite: http.SameSiteLaxMode,
	})
	w.WriteHeader(204)
}

func (a *api) importCSV(w http.ResponseWriter, r *http.Request) {
	user, course := a.uid(r), r.PathValue("id")
	if _, err := a.Course.GetCourse(r.Context(), user, course); err != nil {
		a.fail(w, r, err)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxCSV+1<<20)
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		bad(w, "Send the CSV file as a form.")
		return
	}
	defer r.MultipartForm.RemoveAll()
	f, hdr, err := r.FormFile("file")
	if err != nil {
		bad(w, "Choose a CSV file.")
		return
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxCSV+1))
	if err != nil {
		a.fail(w, r, err)
		return
	}
	if len(data) > maxCSV {
		bad(w, "That file is bigger than 1 MB.")
		return
	}
	j, err := a.Importer.Import(r.Context(), user, course, hdr.Filename, data)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	j.RowErrors = nz(j.RowErrors)
	writeJSON(w, 202, j)
}

func (a *api) getImport(w http.ResponseWriter, r *http.Request) {
	j, err := a.Imports.GetImport(r.Context(), a.uid(r), r.PathValue("id"))
	if err != nil {
		a.fail(w, r, err)
		return
	}
	j.RowErrors = nz(j.RowErrors)
	writeJSON(w, 200, j)
}
