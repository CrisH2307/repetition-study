package app

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func ok(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) }

func TestHardenHeaders(t *testing.T) {
	rec := httptest.NewRecorder()
	Harden(http.HandlerFunc(ok), true).ServeHTTP(rec, httptest.NewRequest("GET", "/v1/me", nil))
	for k, v := range map[string]string{"X-Content-Type-Options": "nosniff", "Cache-Control": "no-store"} {
		if rec.Header().Get(k) != v {
			t.Errorf("%s = %q", k, rec.Header().Get(k))
		}
	}
	if rec.Header().Get("Strict-Transport-Security") == "" || rec.Header().Get("Content-Security-Policy") == "" {
		t.Error("missing HSTS or CSP")
	}
	rec = httptest.NewRecorder()
	Harden(http.HandlerFunc(ok), false).ServeHTTP(rec, httptest.NewRequest("GET", "/v1/me", nil))
	if rec.Header().Get("Strict-Transport-Security") != "" {
		t.Error("HSTS must be off without Secure")
	}
}

func TestRateLimitSignInAndTurns(t *testing.T) {
	t.Setenv("RATE_SIGNIN_PER_MIN", "3")
	t.Setenv("RATE_TURNS_PER_MIN", "2")
	h := Harden(http.HandlerFunc(ok), true)
	code := func(path, ip string) int {
		r := httptest.NewRequest("POST", path, nil)
		r.Header.Set("Fly-Client-IP", ip)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, r)
		return rec.Code
	}
	for i := 0; i < 3; i++ {
		if c := code("/v1/auth/redeem", "1.1.1.1"); c != 200 {
			t.Fatalf("sign-in %d: %d", i, c)
		}
	}
	if c := code("/v1/auth/redeem", "1.1.1.1"); c != 429 {
		t.Errorf("4th sign-in = %d, want 429", c)
	}
	if c := code("/v1/auth/redeem", "2.2.2.2"); c != 200 {
		t.Errorf("other address = %d, want 200", c)
	}
	code("/v1/teach/sessions/x/turns", "3.3.3.3")
	code("/v1/teach/sessions/x/turns", "3.3.3.3")
	if c := code("/v1/teach/sessions/x/turns", "3.3.3.3"); c != 429 {
		t.Errorf("3rd turn = %d, want 429", c)
	}
}

func TestLimiterRefills(t *testing.T) {
	now := time.Now()
	l := newLimiter(1)
	l.now = func() time.Time { return now }
	if !l.allow("a") || l.allow("a") {
		t.Fatal("burst of 1 expected")
	}
	now = now.Add(time.Minute)
	if !l.allow("a") {
		t.Error("should refill after a minute")
	}
}

func TestUploadGuard(t *testing.T) {
	t.Setenv("UPLOADS_PER_DAY", "3")
	t.Setenv("UPLOADS_READING_AT_ONCE", "2")
	t.Setenv("UPLOADS_TOTAL_MB", "1")
	g := newUploadGuard()
	if g.reserve("u", 10) != "" || g.reserve("u", 10) != "" {
		t.Fatal("first two must pass")
	}
	if g.reserve("u", 10) == "" {
		t.Error("third at once must be refused")
	}
	g.release("u")
	if g.reserve("u", 2<<20) == "" {
		t.Error("over total size must be refused")
	}
	if g.reserve("u", 10) != "" {
		t.Error("third of the day must pass")
	}
	g.release("u")
	if g.reserve("u", 10) == "" {
		t.Error("fourth of the day must be refused")
	}
	if g.reserve("other", 10) != "" {
		t.Error("another user is not affected")
	}
}

func TestSignInKeyedPerStudentBehindEdge(t *testing.T) {
	t.Setenv("RATE_SIGNIN_PER_MIN", "10")
	t.Setenv("RATE_SIGNIN_EDGE_PER_MIN", "30")
	h := Harden(http.HandlerFunc(ok), true)
	code := func(edge, student string) int {
		r := httptest.NewRequest("POST", "/v1/auth/redeem", nil)
		r.Header.Set("Fly-Client-IP", edge)
		r.Header.Set("X-Vercel-Forwarded-For", student)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, r)
		return rec.Code
	}
	for i := 0; i < 20; i++ {
		if c := code("9.9.9.9", fmt.Sprintf("10.0.0.%d", i)); c != 200 {
			t.Fatalf("student %d: %d", i, c)
		}
	}
	for i := 0; i < 10; i++ {
		code("8.8.8.8", "10.1.1.1")
	}
	if c := code("8.8.8.8", "10.1.1.1"); c != 429 {
		t.Errorf("11th sign-in by one student = %d, want 429", c)
	}
	// A direct caller rotating the forwarded header is stopped by the edge cap.
	for i := 0; i < 30; i++ {
		code("7.7.7.7", fmt.Sprintf("10.2.0.%d", i))
	}
	if c := code("7.7.7.7", "10.2.9.9"); c != 429 {
		t.Errorf("rotating headers = %d, want 429", c)
	}
}
