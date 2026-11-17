Unit P1-AUTH, wave 2. Branch `feature/invite-auth`. Model: Opus 5.5, effort high. Labels: `User Story`, `phase-1`, `wave-2`.
Files: `internal/auth/`, `cmd/invite/`, `rpi/P1-AUTH/`.
Read first: D29, P1-CONTRACT's impl-log.
1. `cmd/invite -name "Dino"` prints a one-time link. Tokens: 32 random bytes, only the SHA-256 is stored.
2. Redeeming an invite creates the user (or signs in its user) and sets an HttpOnly, Secure (configurable for localhost), SameSite=Lax cookie with a 30-day session. Sign-out deletes the session.
3. Middleware puts the user id in the request context; requests without a valid session get 401 with a plain message. CSRF: state-changing requests need the same-origin check.
4. Tests: expired session, reused invite, tampered cookie, sign-out.
Done when: tests green, `/code-review medium` clean. PR into qa with the labels above. Never merge. Report under 200 words.
