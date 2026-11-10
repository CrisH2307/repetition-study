Unit P0-CONTRACT, wave 0. Branch `feature/contract-v2` from qa. Model: Opus 5.5, effort high. Everything depends on this unit, so be careful and small.
Files: `internal/domain/`, `api/`, `migrations/`, `internal/fakes/`, `go.mod`, `go.sum` (only to add one OpenAPI validator module, verify its real path first), `rpi/P0-CONTRACT/`.
Read first: `docs/improved/strategy-review-fable.md` sections 4.2 and 4.3 and the Phase 0 table, `internal/domain/*.go`, `migrations/0001_core.sql`, `internal/teach/store.go`.

Add, as additive changes only (nothing existing breaks):
1. `TeachStore` interface in `internal/domain`, matching `internal/teach.Store` so P0-STORE can implement it.
2. `Pack` type (sources, concepts, claims, optional history) and a `PackLoader` interface for `cmd/seed`.
3. `GetReport(sessionID)` on the engine contract and `Report.Status` with values `checked` and `unchecked`.
4. An explicit event id: the teach-back event is keyed by session id (idempotency). Add `SessionID` to `ReviewEvent` and a unique partial index so one session writes one `teach_back` event.
5. `DueItem.Reason` (string).
6. Migration `0003`: `source_blocks(id, source_id, page, ordinal, kind, text_md, text_norm, verified_by, verified_at)`, `users.time_zone` (default `America/Toronto`), `review_events` gains nullable `claim_id`, `session_id`, `self_grade`, `checker_model`, `checker_version`. Keep `review_events` append-only and prove the trigger still works.
7. Update `internal/fakes` for every new method. Update `api/openapi.yaml` for the new fields and validate it with a real OpenAPI validator in a Go test.

Not in this unit: claim-level scheduling state (`claim_states`), accounts, jobs table. Those are Phase 1.
Done when: `go build ./...`, `go vet ./...`, `make up migrate test` are green on a clean database; migration applies on top of the existing schema with no data loss; the append-only test still passes.
Do not open a PR. Push the branch and stop. Report under 200 words with every new name added.
