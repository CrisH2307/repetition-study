package main

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// withWeb serves the built web app from dir. API routes keep priority; any other
// GET/HEAD that is not a real file falls back to index.html (SPA routing).
func withWeb(api http.Handler, dir string) http.Handler {
	files := http.FileServer(http.Dir(dir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		if p == "/healthz" || strings.HasPrefix(p, "/v1/") || (r.Method != http.MethodGet && r.Method != http.MethodHead) {
			api.ServeHTTP(w, r)
			return
		}
		if st, err := os.Stat(filepath.Join(dir, filepath.Clean("/"+p))); err != nil || st.IsDir() && p != "/" {
			r.URL.Path = "/"
		}
		files.ServeHTTP(w, r)
	})
}
