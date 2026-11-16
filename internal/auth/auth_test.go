package auth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CrisH2307/repetition-study/internal/domain"
	"github.com/CrisH2307/repetition-study/internal/fakes"
)

func setup() (*Service, *fakes.AuthStore, *fakes.Clock) {
	st, clk := &fakes.AuthStore{}, fakes.NewClock()
	return &Service{Store: st, Clock: clk}, st, clk
}

// spy records the hashes the store receives.
type spy struct {
	domain.AuthStore
	hashes [][]byte
}

func (s *spy) CreateInvite(ctx context.Context, n string, h []byte) (domain.Invite, error) {
	s.hashes = append(s.hashes, h)
	return s.AuthStore.CreateInvite(ctx, n, h)
}

func (s *spy) CreateSession(ctx context.Context, u string, h []byte, e time.Time) error {
	s.hashes = append(s.hashes, h)
	return s.AuthStore.CreateSession(ctx, u, h, e)
}

func TestStoreSeesOnlyHashes(t *testing.T) {
	sp := &spy{AuthStore: &fakes.AuthStore{}}
	s := &Service{Store: sp, Clock: fakes.NewClock()}
	ctx := context.Background()
	inv, _ := s.NewInvite(ctx, "Dino")
	_, sess, exp, err := s.Redeem(ctx, inv)
	if err != nil {
		t.Fatal(err)
	}
	for i, tok := range []string{inv, sess} {
		if raw, _ := base64.RawURLEncoding.DecodeString(tok); len(raw) != 32 {
			t.Fatalf("token %d has %d bytes, want 32", i, len(raw))
		}
		want := sha256.Sum256([]byte(tok))
		if hex.EncodeToString(sp.hashes[i]) != hex.EncodeToString(want[:]) {
			t.Fatalf("store got %x, want sha256 of token", sp.hashes[i])
		}
	}
	if exp != fakes.Epoch.Add(SessionTTL) {
		t.Fatalf("expires %v", exp)
	}
}

func TestExpiredSession(t *testing.T) {
	s, _, clk := setup()
	ctx := context.Background()
	inv, _ := s.NewInvite(ctx, "Dino")
	_, sess, _, _ := s.Redeem(ctx, inv)
	clk.Advance(SessionTTL - time.Second)
	if _, err := s.UserFor(ctx, sess); err != nil {
		t.Fatalf("before expiry: %v", err)
	}
	clk.Advance(time.Second)
	if _, err := s.UserFor(ctx, sess); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("at expiry: got %v, want ErrNotFound", err)
	}
}

func TestReusedInviteSignsInSameUser(t *testing.T) {
	s, st, _ := setup()
	ctx := context.Background()
	inv, _ := s.NewInvite(ctx, "Dino")
	u1, s1, _, _ := s.Redeem(ctx, inv)
	u2, s2, _, err := s.Redeem(ctx, inv)
	if err != nil || u1.ID != u2.ID || s1 == s2 || len(st.Users) != 1 {
		t.Fatalf("reuse: %v %v %v users=%d", err, u1, u2, len(st.Users))
	}
	if _, _, _, err := s.Redeem(ctx, inv+"x"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("unknown invite: %v", err)
	}
	if _, _, _, err := s.Redeem(ctx, ""); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("empty invite: %v", err)
	}
}

func TestEmptyNameRefused(t *testing.T) {
	s, _, _ := setup()
	var ue *domain.UserError
	if _, err := s.NewInvite(context.Background(), ""); !errors.As(err, &ue) {
		t.Fatalf("got %v", err)
	}
}

func handler(s *Service) http.Handler {
	a := HTTP{Auth: s, Secure: true}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/auth/redeem", a.Redeem)
	mux.Handle("POST /v1/auth/sign-out", a.Require(http.HandlerFunc(a.SignOut)))
	mux.Handle("GET /v1/me", a.Require(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, _ := UserFrom(r.Context())
		w.Write([]byte(u.ID))
	})))
	return Protect(mux)
}

func call(h http.Handler, method, path, body, cookie string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	if cookie != "" {
		r.AddCookie(&http.Cookie{Name: CookieName, Value: cookie})
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func redeem(t *testing.T, h http.Handler, inv string) *http.Cookie {
	t.Helper()
	w := call(h, "POST", "/v1/auth/redeem", `{"token":"`+inv+`"}`, "")
	if w.Code != 200 {
		t.Fatalf("redeem: %d %s", w.Code, w.Body)
	}
	c := w.Result().Cookies()[0]
	if c.Name != CookieName || !c.HttpOnly || !c.Secure || c.SameSite != http.SameSiteLaxMode || c.Path != "/" {
		t.Fatalf("cookie flags: %+v", c)
	}
	return c
}

func TestCookieFlow(t *testing.T) {
	s, _, _ := setup()
	h := handler(s)
	inv, _ := s.NewInvite(context.Background(), "Dino")
	c := redeem(t, h, inv)
	if w := call(h, "GET", "/v1/me", "", c.Value); w.Code != 200 || w.Body.String() != "user-1" {
		t.Fatalf("me: %d %s", w.Code, w.Body)
	}
	if w := call(h, "GET", "/v1/me", "", ""); w.Code != 401 || !strings.Contains(w.Body.String(), "sign in") {
		t.Fatalf("no cookie: %d %s", w.Code, w.Body)
	}
	if w := call(h, "POST", "/v1/auth/redeem", `{"token":"nope"}`, ""); w.Code != 404 {
		t.Fatalf("bad invite: %d", w.Code)
	}
}

func TestTamperedCookie(t *testing.T) {
	s, _, _ := setup()
	h := handler(s)
	inv, _ := s.NewInvite(context.Background(), "Dino")
	c := redeem(t, h, inv)
	b := []byte(c.Value)
	b[0] ^= 1
	w := call(h, "GET", "/v1/me", "", string(b))
	if w.Code != 401 {
		t.Fatalf("tampered: %d", w.Code)
	}
	if cl := w.Result().Cookies(); len(cl) != 1 || cl[0].MaxAge >= 0 {
		t.Fatalf("bad cookie not cleared: %+v", cl)
	}
}

func TestSignOut(t *testing.T) {
	s, _, _ := setup()
	h := handler(s)
	inv, _ := s.NewInvite(context.Background(), "Dino")
	c := redeem(t, h, inv)
	w := call(h, "POST", "/v1/auth/sign-out", "", c.Value)
	if w.Code != 204 || w.Result().Cookies()[0].MaxAge >= 0 {
		t.Fatalf("sign-out: %d %+v", w.Code, w.Result().Cookies())
	}
	if w := call(h, "GET", "/v1/me", "", c.Value); w.Code != 401 {
		t.Fatalf("old cookie after sign-out: %d", w.Code)
	}
}

func TestCrossOriginPostRefused(t *testing.T) {
	s, _, _ := setup()
	h := handler(s)
	inv, _ := s.NewInvite(context.Background(), "Dino")
	r := httptest.NewRequest("POST", "http://app.example/v1/auth/redeem", strings.NewReader(`{"token":"`+inv+`"}`))
	r.Header.Set("Origin", "https://evil.example")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Fatalf("cross-origin: %d", w.Code)
	}
	r = httptest.NewRequest("POST", "http://app.example/v1/auth/redeem", strings.NewReader(`{"token":"`+inv+`"}`))
	r.Header.Set("Sec-Fetch-Site", "same-origin")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("same-origin: %d", w.Code)
	}
}
