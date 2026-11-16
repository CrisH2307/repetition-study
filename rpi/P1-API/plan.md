# P1-API plan

1. `Deps` gains Auth, Course, Reviews, Notes, Today, Own. `Auth == nil` is Phase 0 mode (demo user, no v3 routes), kept for old tests.
2. `v3.go`: handlers for every v3 path. Uploads answer 202, read in a goroutine, plain words on failure.
3. Phase 0 routes act as the signed-in user: session, concept and card ownership first (404), due list filtered.
4. `Wire` builds the real pieces. `Build` adds the monthly cap (`MONTHLY_CAP_USD`) on the checker only. Cookie `Secure` unless `COOKIE_SECURE=false`.
5. `cmd/export -disputes`, `make export-disputes`.
6. Tests: every spec path served (fakes), isolation on Postgres (two users, every id), full loop with the fake model, cap, upload refusals.
7. `web/e2e/loop.e2e.ts` signs in a fresh student and builds the course through the API.

Risks: reading uploads stuck after a restart; cap counted in memory; `LoadPack` gap above.
