package app

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/CrisH2307/repetition-study/internal/domain"
)

const maxUpload = 20 << 20 // a PDF is at most 20 MB (D27)

func (a *api) routesV3(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/me", a.me)
	mux.HandleFunc("GET /v1/courses", a.listSubjects)
	mux.HandleFunc("POST /v1/courses", a.createSubject)
	mux.HandleFunc("GET /v1/courses/{id}", a.getCourse)
	mux.HandleFunc("PATCH /v1/courses/{id}", a.updateCourse)
	mux.HandleFunc("DELETE /v1/courses/{id}", a.archiveCourse)
	mux.HandleFunc("GET /v1/courses/{id}/view", a.courseView)
	mux.HandleFunc("POST /v1/courses/{id}/concepts", a.createConcept)
	mux.HandleFunc("GET /v1/courses/{id}/blocks", a.blocks)
	mux.HandleFunc("GET /v1/courses/{id}/uploads", a.listUploads)
	mux.HandleFunc("POST /v1/courses/{id}/uploads", a.createUpload)
	mux.HandleFunc("GET /v1/uploads/{id}", a.getUpload)
	mux.HandleFunc("GET /v1/uploads/{id}/pages", a.listPages)
	mux.HandleFunc("GET /v1/pages/{id}", a.getPage)
	mux.HandleFunc("PATCH /v1/pages/{id}", a.editPage)
	mux.HandleFunc("GET /v1/pages/{id}/image", a.pageImage)
	mux.HandleFunc("POST /v1/pages/{id}/ready", a.markReady)
	mux.HandleFunc("POST /v1/pages/{id}/unready", a.markNotReady)
	mux.HandleFunc("GET /v1/concepts/{id}/claims", a.listClaims)
	mux.HandleFunc("POST /v1/concepts/{id}/claims", a.addClaim)
	mux.HandleFunc("PATCH /v1/claims/{id}", a.editClaim)
	mux.HandleFunc("DELETE /v1/claims/{id}", a.deleteClaim)
	mux.HandleFunc("POST /v1/teach/sessions/{id}/self-grades", a.selfGrades)
	mux.HandleFunc("POST /v1/teach/sessions/{id}/disputes", a.dispute)
	mux.HandleFunc("GET /v1/today", a.today)
	mux.HandleFunc("GET /v1/week", a.week)
}

// nz makes a nil slice an empty one: the spec's arrays are not nullable.
func nz[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

func (a *api) me(w http.ResponseWriter, r *http.Request) {
	u, _ := authUser(r)
	writeJSON(w, 200, u)
}

func (a *api) getCourse(w http.ResponseWriter, r *http.Request) {
	s, err := a.Course.GetCourse(r.Context(), a.uid(r), r.PathValue("id"))
	if err != nil {
		a.fail(w, r, err)
		return
	}
	writeJSON(w, 200, toSubjectOut(s))
}

func (a *api) updateCourse(w http.ResponseWriter, r *http.Request) {
	s, ok := readSubject(w, r)
	if !ok {
		return
	}
	s.ID = r.PathValue("id")
	got, err := a.Course.UpdateCourse(r.Context(), a.uid(r), s)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	writeJSON(w, 200, toSubjectOut(got))
}

func (a *api) archiveCourse(w http.ResponseWriter, r *http.Request) {
	if err := a.Course.ArchiveCourse(r.Context(), a.uid(r), r.PathValue("id")); err != nil {
		a.fail(w, r, err)
		return
	}
	w.WriteHeader(204)
}

func (a *api) courseView(w http.ResponseWriter, r *http.Request) {
	v, err := a.Reviews.CourseView(r.Context(), a.uid(r), r.PathValue("id"))
	if err != nil {
		a.fail(w, r, err)
		return
	}
	cards := nz(v.Cards)
	for i := range cards {
		cards[i].Claims = nz(cards[i].Claims)
	}
	writeJSON(w, 200, map[string]any{ // the course carries exam_date as a plain date, like /v1/courses
		"course": toSubjectOut(v.Course), "notes": nz(v.Notes), "cards": cards, "mistakes": nz(v.Mistakes),
	})
}

func (a *api) createConcept(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name string `json:"name"`
	}
	if !decode(w, r, &in) {
		return
	}
	if strings.TrimSpace(in.Name) == "" {
		bad(w, "name is required")
		return
	}
	c, err := a.Course.CreateConcept(r.Context(), a.uid(r), r.PathValue("id"), strings.TrimSpace(in.Name))
	if err != nil {
		a.fail(w, r, err)
		return
	}
	c.Claims = nz(c.Claims)
	writeJSON(w, 201, c)
}

func (a *api) blocks(w http.ResponseWriter, r *http.Request) {
	bs, err := a.Course.ReadyBlocks(r.Context(), a.uid(r), r.PathValue("id"))
	if err != nil {
		a.fail(w, r, err)
		return
	}
	writeJSON(w, 200, nz(bs))
}

// ---- uploads and pages ----

func (a *api) listUploads(w http.ResponseWriter, r *http.Request) {
	us, err := a.Course.ListUploads(r.Context(), a.uid(r), r.PathValue("id"))
	if err != nil {
		a.fail(w, r, err)
		return
	}
	writeJSON(w, 200, nz(us))
}

// createUpload stores the file and answers 202 at once. Reading runs in the
// background; the page polls the upload until its status is not reading.
func (a *api) createUpload(w http.ResponseWriter, r *http.Request) {
	user, course := a.uid(r), r.PathValue("id")
	if _, err := a.Course.GetCourse(r.Context(), user, course); err != nil {
		a.fail(w, r, err)
		return
	}
	// A slow phone upload needs longer than the server's 15 s read timeout.
	http.NewResponseController(w).SetReadDeadline(time.Now().Add(2 * time.Minute))
	r.Body = http.MaxBytesReader(w, r.Body, maxUpload+1<<20)
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			bad(w, "That file is bigger than 20 MB.")
		} else {
			bad(w, "Send the file or the text as a form.")
		}
		return
	}
	defer r.MultipartForm.RemoveAll()
	name, text := strings.TrimSpace(r.FormValue("filename")), r.FormValue("text")
	var pdf []byte
	if f, _, err := r.FormFile("file"); err == nil {
		defer f.Close()
		if pdf, err = io.ReadAll(io.LimitReader(f, maxUpload+1)); err != nil {
			a.fail(w, r, err)
			return
		}
	}
	switch {
	case name == "":
		bad(w, "Give the upload a name.")
		return
	case len(pdf) > 0 && strings.TrimSpace(text) != "":
		bad(w, "Send a file or pasted text, not both.")
		return
	case len(pdf) == 0 && strings.TrimSpace(text) == "":
		bad(w, "Choose a PDF or paste some text.")
		return
	case len(pdf) > maxUpload:
		bad(w, "That file is bigger than 20 MB.")
		return
	case len(pdf) > 0 && !strings.HasPrefix(string(pdf[:min(len(pdf), 5)]), "%PDF-"):
		bad(w, "That file is not a PDF.")
		return
	}
	u, err := a.Course.CreateUpload(r.Context(), user, domain.Upload{SubjectID: course, Filename: name}, pdf, text)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	go a.readUpload(user, u.ID)
	writeJSON(w, 202, u)
}

// readUpload never panics the server. A failure leaves plain words on the upload.
func (a *api) readUpload(user, id string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute) // ponytail: an upload stuck in reading after a restart stays so; add a sweep if it happens
	defer cancel()
	fail := func(v any) {
		a.Log.Error("reading an upload failed", "upload", id, "err", v)
		a.Course.SetUploadStatus(ctx, user, id, domain.UploadNeedsChecking, "We could not read this file. Try again or paste the text.")
	}
	defer func() {
		if v := recover(); v != nil {
			fail(v)
		}
	}()
	if err := a.Notes.ReadUpload(ctx, user, id); err != nil {
		fail(err)
	}
}

func (a *api) getUpload(w http.ResponseWriter, r *http.Request) {
	u, err := a.Course.GetUpload(r.Context(), a.uid(r), r.PathValue("id"))
	if err != nil {
		a.fail(w, r, err)
		return
	}
	writeJSON(w, 200, u)
}

func (a *api) listPages(w http.ResponseWriter, r *http.Request) {
	ps, err := a.Course.ListPages(r.Context(), a.uid(r), r.PathValue("id"))
	if err != nil {
		a.fail(w, r, err)
		return
	}
	writeJSON(w, 200, nz(ps))
}

func (a *api) getPage(w http.ResponseWriter, r *http.Request) {
	p, err := a.Course.GetPage(r.Context(), a.uid(r), r.PathValue("id"))
	if err != nil {
		a.fail(w, r, err)
		return
	}
	writeJSON(w, 200, p)
}

func (a *api) editPage(w http.ResponseWriter, r *http.Request) {
	var in struct {
		EditedText *string `json:"edited_text"`
	}
	if !decode(w, r, &in) {
		return
	}
	if in.EditedText == nil {
		bad(w, "edited_text is required")
		return
	}
	p, err := a.Notes.EditPage(r.Context(), a.uid(r), r.PathValue("id"), *in.EditedText)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	writeJSON(w, 200, p)
}

func (a *api) pageImage(w http.ResponseWriter, r *http.Request) {
	img, err := a.Course.PageImage(r.Context(), a.uid(r), r.PathValue("id"))
	if err != nil {
		a.fail(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "private, max-age=3600")
	w.Write(img)
}

func (a *api) markReady(w http.ResponseWriter, r *http.Request) {
	p, err := a.Notes.MarkReady(r.Context(), a.uid(r), r.PathValue("id"))
	if err != nil {
		a.fail(w, r, err)
		return
	}
	writeJSON(w, 200, p)
}

func (a *api) markNotReady(w http.ResponseWriter, r *http.Request) {
	p, err := a.Notes.MarkNotReady(r.Context(), a.uid(r), r.PathValue("id"))
	if err != nil {
		a.fail(w, r, err)
		return
	}
	writeJSON(w, 200, p)
}

// ---- claims ----

func (a *api) listClaims(w http.ResponseWriter, r *http.Request) {
	cs, err := a.Course.ListClaims(r.Context(), a.uid(r), r.PathValue("id"))
	if err != nil {
		a.fail(w, r, err)
		return
	}
	writeJSON(w, 200, nz(cs))
}

func (a *api) addClaim(w http.ResponseWriter, r *http.Request) {
	var in domain.ClaimInput
	if !decode(w, r, &in) {
		return
	}
	c, err := a.Notes.AddClaim(r.Context(), a.uid(r), r.PathValue("id"), in)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	writeJSON(w, 201, c)
}

func (a *api) editClaim(w http.ResponseWriter, r *http.Request) {
	var in domain.ClaimInput
	if !decode(w, r, &in) {
		return
	}
	c, err := a.Notes.EditClaim(r.Context(), a.uid(r), r.PathValue("id"), in)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	writeJSON(w, 200, c)
}

func (a *api) deleteClaim(w http.ResponseWriter, r *http.Request) {
	if err := a.Notes.DeleteClaim(r.Context(), a.uid(r), r.PathValue("id")); err != nil {
		a.fail(w, r, err)
		return
	}
	w.WriteHeader(204)
}

// ---- self-grades and disputes ----

func (a *api) selfGrades(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Grades []struct {
			ClaimID string            `json:"claim_id"`
			Rating  domain.SelfRating `json:"rating"`
		} `json:"grades"`
	}
	if !decode(w, r, &in) {
		return
	}
	sid := r.PathValue("id")
	gs := make([]domain.SelfGrade, 0, len(in.Grades))
	for _, g := range in.Grades {
		switch g.Rating {
		case domain.RatingRight, domain.RatingNotSure, domain.RatingWrong:
		default:
			bad(w, "rating must be right, not_sure or wrong")
			return
		}
		gs = append(gs, domain.SelfGrade{SessionID: sid, ClaimID: g.ClaimID, Rating: g.Rating})
	}
	if err := a.Reviews.SaveSelfGrades(r.Context(), a.uid(r), sid, gs); err != nil {
		a.fail(w, r, err)
		return
	}
	w.WriteHeader(204)
}

func (a *api) dispute(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ClaimID *string `json:"claim_id"`
		Note    string  `json:"note"`
	}
	if !decode(w, r, &in) {
		return
	}
	d, err := a.Reviews.CreateDispute(r.Context(), a.uid(r),
		domain.Dispute{SessionID: r.PathValue("id"), ClaimID: in.ClaimID, Note: strings.TrimSpace(in.Note)})
	if err != nil {
		a.fail(w, r, err)
		return
	}
	writeJSON(w, 201, d)
}

// ---- today and week ----

func (a *api) today(w http.ResponseWriter, r *http.Request) {
	now := a.Clock.Now().UTC()
	st, err := a.Reviews.StudyState(r.Context(), a.uid(r), now.AddDate(0, 0, -8)) // covers the start of any local week
	if err != nil {
		a.fail(w, r, err)
		return
	}
	t, err := a.Today.Today(st, now)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	writeJSON(w, 200, nzToday(t))
}

func (a *api) week(w http.ResponseWriter, r *http.Request) {
	ws, err := parseDate(r.URL.Query().Get("week"))
	if err != nil {
		bad(w, "week must be YYYY-MM-DD")
		return
	}
	now := a.Clock.Now().UTC()
	st, err := a.Reviews.StudyState(r.Context(), a.uid(r), ws.AddDate(0, 0, -2))
	if err != nil {
		a.fail(w, r, err)
		return
	}
	p, err := a.Today.Week(st, ws, now)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	p.Courses, p.Days, p.Notes = nz(p.Courses), nz(p.Days), nz(p.Notes)
	for i := range p.Days {
		p.Days[i].Items = nz(p.Days[i].Items)
	}
	writeJSON(w, 200, p)
}

func nzToday(t domain.Today) domain.Today {
	t.Courses, t.StudyTime = nz(t.Courses), nz(t.StudyTime)
	for i := range t.Courses {
		t.Courses[i].Items = nz(t.Courses[i].Items)
	}
	return t
}
