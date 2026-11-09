You own P0-INT on branch feature/p0-integration, from qa after P0-STORE,
P0-OUTAGE, P0-SRS, P0-TZ and P0-LLM are merged. Owner: Cris with Claude. Files:
cmd/api/, cmd/seed/, internal/app/, Makefile, compose.yaml, go.mod, go.sum,
scripts/seed.sql, rpi/P0-INT/. This replaces rpi/INT for forward work.

Read first: docs/STATUS.md (F4, F14), docs/DECISIONS.md, docs/ROADMAP.md,
rpi/INT/PROMPT.md (assertion list). Decisions: D24; D23 (the unchecked assertion in step 4 assumes it is Decided);
D7 (the fixture in step 2 exists only after P0-HYG finishes its D7 step);
D11 (step 5 real run stays inside the model budget). Write plan.md, then STOP
until Cris approves it (regulation 5). If a unit has a bug, report it; do not fix
it here.

Steps
1. internal/app: composition root (pgxpool, real scheduler, planner, checker,
   teach engine with the Postgres store, LLM clients from env). Add pgxpool to
   go.mod; verify the version.
2. cmd/seed: load a pack file (the fixture, or content/private/ when present)
   into sources, concepts and claims; optional history so the demo concept is
   due about 7 days out.
3. Handlers for POST teach sessions, turns (SSE), finish, GET report, GET
   reviews/due, POST cards review, POST concepts override, matching
   api/openapi.yaml. Server timeouts on http.Server (F14).
4. One automated full-loop test with fakes: seed; teach with one planted mistake;
   Finish; assert a verified citation, an earlier next review, a teach_back
   event, a gap card. Assert an unchecked report writes no event.
5. One run with real models, recorded in impl-log.md.

Tests: the full-loop test, handler tests with httptest, go test -race ./...

Done when: full-loop test green with fakes and one real run recorded.

Report in under 200 words: done, tests with results, deviations from the contract.
