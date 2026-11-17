# P3-HARDEN plan
1. `internal/app/middleware.go`: headers, token-bucket rate limits, upload guard; wired in `cmd/api` (Harden) and `createUpload`/`readUpload`.
2. Contract (approved by main session): `AuthStore.DeleteUserSessions`, `RevokeInvite`; `Auth.SignOutEverywhere`, `RevokeInvite`; `domain.SessionIdleTTL`. Migration 0007 adds `invites.revoked_at`, `auth_sessions.last_seen_at`.
3. `cmd/invite -revoke`; `sign-out?all=true` (no openapi change, api/ is off limits).
4. Mark each line in docs/SECURITY.md.
Risks: counters in memory; store test needs DATABASE_URL.
