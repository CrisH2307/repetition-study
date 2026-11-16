package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWithWeb(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "index.html"), []byte("INDEX"), 0o644)
	os.MkdirAll(filepath.Join(dir, "assets"), 0o755)
	os.WriteFile(filepath.Join(dir, "assets", "a.js"), []byte("JS"), 0o644)
	api := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("API")) })
	h := withWeb(api, dir)
	for path, want := range map[string]string{"/": "INDEX", "/teach/3": "INDEX", "/assets/a.js": "JS", "/healthz": "API", "/v1/subjects": "API", "/assets": "INDEX"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if got := strings.TrimSpace(w.Body.String()); got != want {
			t.Errorf("%s: got %q want %q", path, got, want)
		}
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("POST", "/anything", nil))
	if w.Body.String() != "API" {
		t.Error("non-GET must go to the API")
	}
}
