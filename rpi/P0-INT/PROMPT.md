Unit P0-INT, wave 2, after P0-CONTRACT, P0-STORE, P0-SRS, P0-TZ, P0-CITE and P0-OUTAGE are merged into qa. Branch `feature/integration`. Model: Sonnet 5.5, effort high. Move to Opus 5.5 if a test keeps failing after two different fixes.
Files: `cmd/api/`, `cmd/seed/`, `internal/app/`, `Makefile`, `compose.yaml`, `go.mod`, `go.sum`, `rpi/P0-INT/`.
Read first: `docs/PLAN.md`, `docs/improved/strategy-review-fable.md` section 4.1 and the Phase 0 table, `api/openapi.yaml`, `internal/domain/`.
If a unit has a bug, report it. Do not patch another unit's files.

Steps
1. `internal/app`: composition root. Real implementations replace fakes in this order: scheduler, planner, checker, teach engine, store. After each, `make test`. Auth is the seeded demo user. Server read and write timeouts are set.
2. Handlers for: teach sessions, turns (SSE `token`, `done`, `error`), finish, report, due, card review, override, subjects. Skip digest and plan editing.
3. `cmd/seed`: load a pack through the pack loader. The committed pack uses the team fixture. If `content/private/` exists, load the real source text from it.
4. One automated full-loop test with the fake model: seed, teach the demo concept with one planted mistake, finish. Assert: a verified citation, an earlier next review for the concept, a `teach_back` event, a new gap card, and a changed weekly budget for that subject. Repeat Finish and assert one event only.
5. The same test once with real models, behind the build tag `realllm`, only if keys exist. Otherwise say it is unverified.
6. Contract test: handlers match `api/openapi.yaml`.

Done when: `make up migrate seed test` is green on a clean database; the loop test passes; with the checker failing, the report is `unchecked` and no event is written.
Push the branch, then open a PR into qa with `gh pr create --base qa` (body: "Closes #<issue>", tests run, what is unverified). Never merge. Stop. Report under 200 words, listing every deviation from the contract.

Contract note (from the P0-CONTRACT report, PR #28): use `domain.PackLoader` for `cmd/seed`, `ReportReader.GetReport` for the report endpoint, and `SessionScheduler.ApplyTeachBackSession` for Finish. These are separate interfaces, not new methods on `TeachEngine` and `Scheduler`.
