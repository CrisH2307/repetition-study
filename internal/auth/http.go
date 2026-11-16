package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/CrisH2307/repetition-study/internal/domain"
)

// CookieName is the session cookie (api/openapi.yaml securitySchemes).
const CookieName = "session"

// HTTP holds the auth handlers and middleware. Secure is false only for
// plain-http localhost; everywhere else the cookie needs HTTPS.
type HTTP struct {
	Auth   domain.Auth
	Secure bool
}

type ctxKey struct{}

// UserFrom returns the signed-in user that Require put in the context.
func UserFrom(ctx context.Context) (domain.User, bool) {
	u, ok := ctx.Value(ctxKey{}).(domain.User)
	return u, ok
}

// Protect refuses cross-origin POST, PATCH, PUT and DELETE (CSRF). Wrap the
// whole API with it, redeem included. Uses the browser's Sec-Fetch-Site or
// Origin header; requests with neither (curl, tests) pass.
func Protect(h http.Handler) http.Handler {
	return http.NewCrossOriginProtection().Handler(h)
}

// Require answers 401 unless the request has a valid session cookie, and puts
// the user in the context otherwise.
func (a HTTP) Require(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(CookieName)
		if err != nil {
			writeErr(w, http.StatusUnauthorized, "Please sign in with your invite link.")
			return
		}
		u, err := a.Auth.UserFor(r.Context(), c.Value)
		if errors.Is(err, domain.ErrNotFound) {
			a.clear(w)
			writeErr(w, http.StatusUnauthorized, "Your sign-in has ended. Please open your invite link again.")
			return
		}
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "Could not check your sign-in. Please try again.")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, u)))
	})
}

// Redeem serves POST /v1/auth/redeem: {"token": "..."} to the user and cookie.
func (a HTTP) Redeem(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<10)).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "The invite link is not readable.")
		return
	}
	u, tok, exp, err := a.Auth.Redeem(r.Context(), body.Token)
	if errors.Is(err, domain.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "That invite link is not valid. Ask Cris for a new one.")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Could not sign you in. Please try again.")
		return
	}
	a.set(w, tok, exp)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(u)
}

// SignOut serves POST /v1/auth/sign-out (behind Require).
func (a HTTP) SignOut(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(CookieName); err == nil {
		if err := a.Auth.SignOut(r.Context(), c.Value); err != nil {
			writeErr(w, http.StatusInternalServerError, "Could not sign you out. Please try again.")
			return
		}
	}
	a.clear(w)
	w.WriteHeader(http.StatusNoContent)
}

func (a HTTP) set(w http.ResponseWriter, tok string, exp time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name: CookieName, Value: tok, Path: "/", Expires: exp,
		HttpOnly: true, Secure: a.Secure, SameSite: http.SameSiteLaxMode,
	})
}

func (a HTTP) clear(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name: CookieName, Value: "", Path: "/", MaxAge: -1,
		HttpOnly: true, Secure: a.Secure, SameSite: http.SameSiteLaxMode,
	})
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
