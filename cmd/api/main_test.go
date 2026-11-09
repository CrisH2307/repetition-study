package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthz(t *testing.T) {
	rec := httptest.NewRecorder()
	newMux().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"ok"`) {
		t.Fatalf("got %d %q", rec.Code, rec.Body.String())
	}
}
