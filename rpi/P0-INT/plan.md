# P0-INT plan

1. go.mod: add pgxpool.
2. internal/app: `Wire` (real scheduler, planner, checker, teach engine, store), `NewHandler` (routes), `Build` (pool, models, server with timeouts).
3. Handlers: subjects (list, create), plan generate, further-planning, teach start/turn (SSE)/finish/report, due, card review, concept override. Skipped: digest, plan read/edit, card edit.
4. cmd/api uses `app.Build`. cmd/seed loads a pack through `PackLoader`. `make seed` runs it.
5. Tests: handlers vs openapi.yaml (fakes), full loop on Postgres with fake models, checker-down test, seed test, realllm loop (compiled only).
Risks: none open after P0-OUTAGE merged.
