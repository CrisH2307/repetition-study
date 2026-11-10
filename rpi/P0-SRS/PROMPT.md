Unit P0-SRS, wave 1, after P0-CONTRACT is merged into qa. Branch `fix/srs-idempotency`. Model: Sonnet 5.5, effort medium.
Files: `internal/srs/`, `cmd/migrate/`, `rpi/P0-SRS/`.
Read first: `docs/improved/strategy-review-fable.md` section 4.2 (F7, F12), `internal/srs/`, `cmd/migrate/main.go`, `rpi/U4/research.md`.

Steps
1. Idempotency: the teach-back event is keyed by session id. `ApplyTeachBack` called twice for one session writes one `review_events` row and returns the same event. Use the unique index from P0-CONTRACT.
2. `cmd/migrate` takes a Postgres advisory lock so two runs cannot race.
3. srs tests run in a throwaway schema or a rolled-back transaction and leave no rows behind.
4. Keep the benchmark under 20 ms per 1,000 concepts. Do not start claim-level state: that is Phase 1.

Done when: a test retries Finish after a simulated restart and sees exactly one event; after `go test ./internal/srs/...` the dev database has no new rows; the append-only trigger test still passes.
Push the branch, then open a PR into qa with `gh pr create --base qa` (body: "Closes #<issue>", tests run, what is unverified). Never merge. Stop. Report under 200 words.

Contract note (from the P0-CONTRACT report, PR #28): idempotent teach-back is the new interface `SessionScheduler.ApplyTeachBackSession` in `internal/domain`, keyed by session id, with the unique index `review_events_teach_back_session_uniq`. Implement that interface in `internal/srs`. The old `Scheduler.ApplyTeachBack` stays as it is.
