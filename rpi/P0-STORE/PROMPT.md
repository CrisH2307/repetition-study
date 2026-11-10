Unit P0-STORE, wave 1, after P0-CONTRACT is on integration/p0. Branch `feature/pg-store`. Model: Sonnet 5.5, effort medium.
Files: `internal/store/`, `rpi/P0-STORE/`.
Read first: `docs/improved/strategy-review-fable.md` section 4.2 (F3, F4), `internal/domain/` (the new `TeachStore` and `Pack` types), `internal/teach/store.go`, `migrations/`.

Problem: the teach engine only has an in-memory store, so sessions vanish on restart and nothing is written to Postgres.

Steps
1. Implement the domain `TeachStore` with pgx against `teach_sessions`, `teach_turns`, `taught_claims`, `claim_labels`. Use parameterised SQL only.
2. `LoadConcept` returns only sources approved for that concept's subject.
3. Implement `LoadPack` for the contract's pack loader so `cmd/seed` (written by P0-INT) can fill `sources`, `source_blocks`, `concepts`, `claims`.
4. Tests run inside a throwaway schema or a rolled-back transaction, so a test run leaves no rows in the dev database. Skip cleanly with a clear message when `DATABASE_URL` is unset.
5. Same conformance test suite runs against `MemStore` and the Postgres store.

Done when: a test starts a session, restarts the store, and Finish still works; `claim_labels` rows exist after Finish; `go test -race ./internal/store/...` is green after `make up migrate`.
Do not open a PR. Push the branch and stop. Report under 200 words.
