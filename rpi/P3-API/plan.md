# P3-API plan
- Deps.Sharing (optional); internal/app/v5.go serves the four share paths inside the sign-in wall, preview included. Wire builds share.Service.
- SessionUser selects is_instructor, so /v1/me reports it.
- Remove the four v5 paths from notServed. TestIsolationV5 (real DB) and a fake-stack loop step cover them. Go-backed Playwright share test added.
- Harden (headers, rate limits) still wraps everything in cmd/api; unchanged.
- Note: the spec says GET /v1/shared/{token} has `security: []`. The API now needs sign-in; api/ is outside this unit, so the spec line is stale (follow-up for Cris).
