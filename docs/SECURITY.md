# Security checklist (P3-LOAD, D53, gate G3)

Checked against the code on branch `chore/load-security` on 2026-11-16. Each line says what was checked and **pass** or **gap**. A gap is a finding for Cris; this unit fixed no code.

| # | Area | Result | What was checked |
|---|------|--------|------------------|
| 1 | Cookie flags | pass | `session` cookie is HttpOnly, SameSite=Lax, Secure unless `COOKIE_SECURE=false` (`internal/auth/http.go`). `fly.toml` sets it true. |
| 2 | Session token storage | pass | Invites and sessions are stored as SHA-256 hashes, never raw (`internal/auth/auth.go`). |
| 3 | Session length | pass | Fixed in P3-HARDEN. Sessions expire after 14 idle days (`domain.SessionIdleTTL`, `last_seen_at`); `POST /v1/auth/sign-out?all=true` ends all of a user's sessions. Tests `TestIdleSessionExpires`, `TestSignOutEverywhereAndRevoke`, `TestSignOutAllQuery`, `TestAuthStore` |
| 4 | Invite links | pass | Fixed in P3-HARDEN. `go run ./cmd/invite -revoke <link>` ends the invite and its sessions. Tests `TestRevokeByLink`, `TestSignOutEverywhereAndRevoke`, `TestAuthStore` |
| 5 | CSRF | pass | `http.CrossOriginProtection` wraps the whole API, sign-in included (`auth.Protect`); test `TestCrossOriginPostRefused`. SameSite=Lax is a second layer. |
| 6 | Isolation between users | pass | Tests `TestIsolation`, `TestIsolationV4`, `TestOtherUsersCourse`, `TestCourseLoopAndIsolation`. Unverified: no test was written for every route. |
| 7 | Secrets in git | pass | `.env` is gitignored; keys come from the environment; a search for `sk-or-`, `sk_live`, `sk_test`, `whsec_`, `AKIA`, private keys found none outside docs. |
| 8 | Request body limits | pass | Sign-in body 1 KB, other JSON 1 MiB, upload 20 MB plus form overhead, PDF must start with `%PDF-`. |
| 9 | Upload count and disk | pass | Fixed in P3-HARDEN. Per user: 20 uploads a day, 2 reading at once, 200 MB total (env `UPLOADS_PER_DAY`, `UPLOADS_READING_AT_ONCE`, `UPLOADS_TOTAL_MB`); 429 with plain words. Counts are in memory and reset on restart. Test `TestUploadGuard` |
| 10 | Poppler time limits | pass | `pdftotext` 60 s, each `pdftoppm` page 30 s, whole read 10 min, 200 page cap (`internal/course/pdf.go`). |
| 11 | Poppler load | pass | Fixed in P3-HARDEN. A global queue allows `POPPLER_CONCURRENT` (2) reads at once; the rest wait with status reading. No per-process memory limit. Covered by the semaphore in `readUpload`; unverified under real load |
| 12 | Container user | pass | Image runs as non-root `USER app`. |
| 13 | Server timeouts | pass | Header 5 s, read 15 s, write 30 s, idle 2 min (`internal/app/wire.go`). |
| 14 | Rate limits | pass | Fixed in P3-HARDEN. Token buckets in memory: sign-in per student address forwarded by Vercel (`RATE_SIGNIN_PER_MIN` 10) plus a larger cap per Fly address (`RATE_SIGNIN_EDGE_PER_MIN` 120), writes per session (`RATE_WRITES_PER_MIN` 120), teach turns per session (`RATE_TURNS_PER_MIN` 30); 429 with plain words. Tests `TestRateLimitSignInAndTurns`, `TestSignInKeyedPerStudentBehindEdge`, `TestLimiterRefills` |
| 15 | Webhook signature | n/a | Moot: billing is deferred, no webhook exists. Re-check if P3-BILLING is ever built |
| 16 | Browser security headers | pass | Fixed in P3-HARDEN. Every API reply sets `X-Content-Type-Options: nosniff`, a strict CSP, `Cache-Control: no-store`, `Referrer-Policy`, and HSTS when the cookie is Secure. Test `TestHardenHeaders` |
| 17 | Student text as data | pass | Rule in `CLAUDE.md`; covered by the injection tests (D24). Not re-tested here. |
| 18 | Logs | unverified | A search found no code logging tokens. Fly's own request logs may hold URLs; invite tokens sit after `#` so they are not in URLs (D29). |

## Load result for G3 (local, fake model)

Tool: `go run ./cmd/loadtest` (refuses any host but localhost, or one named with `-allow-host`). Details and numbers: `rpi/P3-LOAD/impl-log.md`. **The bar (p95 under 50 ms at 500 users) was not met on a laptop** where the test, the API and Postgres share 10 cores. Treat it as a first reading, not proof, until it is repeated on a staging copy.

## Open after P3-HARDEN

Gaps 3, 4, 9, 11, 14 and 16 are fixed. Gap 15 is moot. Still open: lines 6 and 18 are unverified. Rate limit and upload counters are per process (fly machine); share them if the app ever runs on more than one machine. The store test for gaps 3 and 4 needs `DATABASE_URL` and was skipped on the machine that wrote it.
