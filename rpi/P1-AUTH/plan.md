# P1-AUTH plan

1. internal/auth/auth.go: `Service` implements domain.Auth. 32-byte tokens, base64url; store sees SHA-256 only. Session 30 days.
2. internal/auth/http.go: `HTTP{Auth, Secure}` with `Redeem`, `SignOut`, `Require` (401 plain words, clears a bad cookie), `UserFrom(ctx)`, `Protect` (CSRF). Cookie `session`: HttpOnly, SameSite=Lax, Path=/, Secure configurable.
3. cmd/invite: `-name`, `APP_URL`, prints `<APP_URL>/invite#<token>` (fragment keeps token out of logs).
Tests: hashes only, expired session, reused invite, tampered cookie, sign-out, cross-origin refused.
Risks: reusable invite link is a long-lived bearer credential (contract choice). Web /invite route is P1-WEB's.
