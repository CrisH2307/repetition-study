Unit P0-SRS, wave 1, after P0-CONTRACT is on integration/p0. Branch `fix/srs-idempotency`. Model: Sonnet 5.5, effort medium.
Files: `internal/srs/`, `cmd/migrate/`, `rpi/P0-SRS/`.
Read first: `docs/improved/strategy-review-fable.md` section 4.2 (F7, F12), `internal/srs/`, `cmd/migrate/main.go`, `rpi/U4/research.md`.

Steps
1. Idempotency: the teach-back event is keyed by session id. `ApplyTeachBack` called twice for one session writes one `review_events` row and returns the same event. Use the unique index from P0-CONTRACT.
2. `cmd/migrate` takes a Postgres advisory lock so two runs cannot race.
3. srs tests run in a throwaway schema or a rolled-back transaction and leave no rows behind.
4. Keep the benchmark under 20 ms per 1,000 concepts. Do not start claim-level state: that is Phase 1.

Done when: a test retries Finish after a simulated restart and sees exactly one event; after `go test ./internal/srs/...` the dev database has no new rows; the append-only trigger test still passes.
Do not open a PR. Push the branch and stop. Report under 200 words.
