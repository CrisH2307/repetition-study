package app

import (
	"net/http"
)

// routesV5 serves pack sharing (D50, D51). Every route is behind sign-in,
// the preview included: there are no public reads.
func (a *api) routesV5(mux *http.ServeMux) {
	if a.Sharing == nil {
		return
	}
	mux.HandleFunc("POST /v1/courses/{id}/share", a.createShare)
	mux.HandleFunc("DELETE /v1/shares/{id}", a.revokeShare)
	mux.HandleFunc("GET /v1/shared/{token}", a.previewShare)
	mux.HandleFunc("POST /v1/shared/{token}/import", a.importShare)
}

func (a *api) createShare(w http.ResponseWriter, r *http.Request) {
	var in struct {
		IncludeSources bool `json:"include_sources"`
	}
	if r.ContentLength != 0 && !decode(w, r, &in) {
		return
	}
	n, err := a.Sharing.Create(r.Context(), a.uid(r), r.PathValue("id"), in.IncludeSources)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store") // the token is shown once
	writeJSON(w, 201, n)
}

func (a *api) revokeShare(w http.ResponseWriter, r *http.Request) {
	if err := a.Sharing.Revoke(r.Context(), a.uid(r), r.PathValue("id")); err != nil {
		a.fail(w, r, err)
		return
	}
	w.WriteHeader(204)
}

func (a *api) previewShare(w http.ResponseWriter, r *http.Request) {
	p, err := a.Sharing.Preview(r.Context(), r.PathValue("token"))
	if err != nil {
		a.fail(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, 200, p)
}

func (a *api) importShare(w http.ResponseWriter, r *http.Request) {
	var in struct {
		SubjectID string `json:"subject_id"`
	}
	if !decode(w, r, &in) {
		return
	}
	if in.SubjectID == "" {
		bad(w, "subject_id is required.")
		return
	}
	out, err := a.Sharing.Import(r.Context(), a.uid(r), r.PathValue("token"), in.SubjectID)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	out.NeedsQuote = nz(out.NeedsQuote)
	writeJSON(w, 200, out)
}
