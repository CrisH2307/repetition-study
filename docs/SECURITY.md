# Security checklist (P3-LOAD, D53, gate G3)

Checked against the code on branch `chore/load-security` on 2026-11-16. Each line says what was checked and **pass** or **gap**. A gap is a finding for Cris; this unit fixed no code.

| # | Area | Result | What was checked |
|---|------|--------|------------------|
| 1 | Cookie flags | pass | `session` cookie is HttpOnly, SameSite=Lax, Secure unless `COOKIE_SECURE=false` (`internal/auth/http.go`). `fly.toml` sets it true. |
| 2 | Session token storage | pass | Invites and sessions are stored as SHA-256 hashes, never raw (`internal/auth/auth.go`). |
| 3 | Session length | gap | A session lasts 30 days and is not renewed or listed. A stolen cookie works for a month. Sign-out only ends the one session. |
| 4 | Invite links | gap | By design (D35) a link works again and again, like a password. No way to revoke one from the app. |
| 5 | CSRF | pass | `http.CrossOriginProtection` wraps the whole API, sign-in included (`auth.Protect`); test `TestCrossOriginPostRefused`. SameSite=Lax is a second layer. |
| 6 | Isolation between users | pass | Tests `TestIsolation`, `TestIsolationV4`, `TestOtherUsersCourse`, `TestCourseLoopAndIsolation`. Unverified: no test was written for every route. |
| 7 | Secrets in git | pass | `.env` is gitignored; keys come from the environment; a search for `sk-or-`, `sk_live`, `sk_test`, `whsec_`, `AKIA`, private keys found none outside docs. |
| 8 | Request body limits | pass | Sign-in body 1 KB, other JSON 1 MiB, upload 20 MB plus form overhead, PDF must start with `%PDF-`. |
| 9 | Upload count and disk | gap | There is no limit on how many uploads one user can make, or at once. A signed-in user could fill the database with 20 MB files. |
| 10 | Poppler time limits | pass | `pdftotext` 60 s, each `pdftoppm` page 30 s, whole read 10 min, 200 page cap (`internal/course/pdf.go`). |
| 11 | Poppler load | gap | Reading runs in an unbounded background goroutine per upload. Many uploads at once run many poppler processes on one machine. No memory limit per process. |
| 12 | Container user | pass | Image runs as non-root `USER app`. |
| 13 | Server timeouts | pass | Header 5 s, read 15 s, write 30 s, idle 2 min (`internal/app/wire.go`). |
| 14 | Rate limits | gap | None anywhere. Sign-in, writes and model calls are all unlimited per user or per address. The monthly money cap (D33) limits spend, not abuse. Invite tokens are random, so guessing is not realistic, but flooding is. |
| 15 | Webhook signature | gap | Not built yet (P3-BILLING, off by default, D49). Re-check this line when that unit merges: HMAC-SHA256 of `t.payload`, 5 minute tolerance, constant-time compare, one event handled once. |
| 16 | Browser security headers | gap | The API sets no `X-Content-Type-Options`, no HSTS and no CSP of its own. Unverified whether Vercel adds them. Most API replies lack `Cache-Control: no-store`; only the export and the page image set one. |
| 17 | Student text as data | pass | Rule in `CLAUDE.md`; covered by the injection tests (D24). Not re-tested here. |
| 18 | Logs | unverified | A search found no code logging tokens. Fly's own request logs may hold URLs; invite tokens sit after `#` so they are not in URLs (D29). |

## Load result for G3 (local, fake model)

Tool: `go run ./cmd/loadtest` (refuses any host but localhost, or one named with `-allow-host`). Details and numbers: `rpi/P3-LOAD/impl-log.md`. **The bar (p95 under 50 ms at 500 users) was not met on a laptop** where the test, the API and Postgres share 10 cores. Treat it as a first reading, not proof, until it is repeated on a staging copy.

## Open in this unit

Gaps 3, 4, 9, 11, 14, 15, 16 need code or settings changes outside this unit's files. They are listed here, not fixed.
