package app

import (
	"crypto/sha256"
	"encoding/hex"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// envInt reads a positive whole number from the environment, else def.
func envInt(name string, def int) int {
	if n, err := strconv.Atoi(os.Getenv(name)); err == nil && n > 0 {
		return n
	}
	return def
}

// bucket is a token bucket: it holds up to burst tokens and refills per minute.
type bucket struct {
	tokens float64
	last   time.Time
}

// limiter is an in-memory token bucket per key. ponytail: per process, so it
// resets on restart and is not shared between machines; add a shared store if
// the app ever runs on more than one.
type limiter struct {
	mu     sync.Mutex
	perMin float64
	burst  float64
	m      map[string]*bucket
	now    func() time.Time
}

func newLimiter(perMin int) *limiter {
	return &limiter{perMin: float64(perMin), burst: float64(perMin), m: map[string]*bucket{}, now: time.Now}
}

func (l *limiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	if len(l.m) > 10000 { // drop buckets that have refilled completely
		for k, b := range l.m {
			if now.Sub(b.last) > time.Minute {
				delete(l.m, k)
			}
		}
	}
	b := l.m[key]
	if b == nil {
		b = &bucket{tokens: l.burst, last: now}
		l.m[key] = b
	}
	b.tokens = min(l.burst, b.tokens+now.Sub(b.last).Minutes()*l.perMin)
	b.last = now
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// clientIP prefers Fly's own header, which a client cannot set on Fly.
func clientIP(r *http.Request) string {
	if ip := r.Header.Get("Fly-Client-IP"); ip != "" {
		return ip
	}
	h, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return h
}

// Harden wraps the API with security headers (gap 16) and rate limits (gap
// 14). Limits per minute come from RATE_SIGNIN_PER_MIN (10, per address),
// RATE_WRITES_PER_MIN (120) and RATE_TURNS_PER_MIN (30), both per session.
func Harden(next http.Handler, secure bool) http.Handler {
	signin := newLimiter(envInt("RATE_SIGNIN_PER_MIN", 10))
	writes := newLimiter(envInt("RATE_WRITES_PER_MIN", 120))
	turns := newLimiter(envInt("RATE_TURNS_PER_MIN", 30))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		h.Set("Cache-Control", "no-store")
		h.Set("Referrer-Policy", "no-referrer")
		if secure {
			h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		ok := true
		switch {
		case r.URL.Path == "/v1/auth/redeem":
			ok = signin.allow(clientIP(r))
		case r.Method != http.MethodGet && r.Method != http.MethodHead && strings.HasPrefix(r.URL.Path, "/v1/"):
			key := clientIP(r)
			if c, err := r.Cookie("session"); err == nil {
				s := sha256.Sum256([]byte(c.Value))
				key = hex.EncodeToString(s[:8])
			}
			ok = writes.allow(key)
			if ok && strings.HasSuffix(r.URL.Path, "/turns") {
				ok = turns.allow(key)
			}
		}
		if !ok {
			h.Set("Content-Type", "application/json")
			h.Set("Retry-After", "60")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error":"You are going a little fast. Please wait a minute and try again."}` + "\n"))
			return
		}
		next.ServeHTTP(w, r)
	})
}

// uploadGuard caps what one user can upload (gap 9) and how many PDFs are
// read at once on the machine (gap 11). ponytail: counts live in memory and
// reset on restart; move them to the store if abuse shows up.
type uploadGuard struct {
	mu      sync.Mutex
	perDay  int
	reading int // per user, at once
	totalB  int64
	users   map[string]*userUploads
	poppler chan struct{} // global cap on concurrent reads; the rest queue
	now     func() time.Time
}

type userUploads struct {
	times   []time.Time
	reading int
	bytes   int64
}

func newUploadGuard() *uploadGuard {
	return &uploadGuard{
		perDay:  envInt("UPLOADS_PER_DAY", 20),
		reading: envInt("UPLOADS_READING_AT_ONCE", 2),
		totalB:  int64(envInt("UPLOADS_TOTAL_MB", 200)) << 20,
		users:   map[string]*userUploads{},
		poppler: make(chan struct{}, envInt("POPPLER_CONCURRENT", 2)),
		now:     time.Now,
	}
}

// reserve admits one upload of n bytes, or returns plain words saying why not.
// Call release when its reading ends.
func (g *uploadGuard) reserve(user string, n int64) string {
	g.mu.Lock()
	defer g.mu.Unlock()
	u := g.users[user]
	if u == nil {
		u = &userUploads{}
		g.users[user] = u
	}
	cut, kept := g.now().Add(-24*time.Hour), u.times[:0]
	for _, t := range u.times {
		if t.After(cut) {
			kept = append(kept, t)
		}
	}
	u.times = kept
	switch {
	case len(u.times) >= g.perDay:
		return "You have reached today's upload limit. Please try again tomorrow."
	case u.reading >= g.reading:
		return "Two files are still being read. Please wait for them to finish."
	case u.bytes+n > g.totalB:
		return "You have reached the total upload size limit. Ask Cris to raise it."
	}
	u.times = append(u.times, g.now())
	u.reading++
	u.bytes += n
	return ""
}

func (g *uploadGuard) release(user string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if u := g.users[user]; u != nil && u.reading > 0 {
		u.reading--
	}
}

var uploads = newUploadGuard()
