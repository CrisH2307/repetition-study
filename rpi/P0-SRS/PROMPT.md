You own P0-SRS on branch fix/srs-idempotency, from qa. Owner: Claude. Files:
internal/srs/, cmd/migrate/, rpi/P0-SRS/.

Read first: docs/STATUS.md (F7, F12), docs/DECISIONS.md, docs/ROADMAP.md.
Decisions: D24. Step 3 waits for P0-CONTRACT to be merged. Write plan.md, then
STOP until Cris approves it (regulation 5).

Steps
1. cmd/migrate: take a Postgres advisory lock for the whole run, release at end.
2. srs tests: run in a throwaway schema (or database) per test run so append-only
   review_events rows never stay in the dev DB.
3. After P0-CONTRACT: take the event id from the new parameter; keep WithEventID
   only as a deprecated shim if callers still use it.

Tests
- Two concurrent migrate runs: both succeed, each file applied once.
- After go test ./internal/srs/ the dev DB has the same review_events count as
  before.
- Retried event with the same id writes once (existing test stays green).
- Benchmark still under 20 ms per 1,000 concepts.

Done when: all pass with Postgres and -race.

Report in under 200 words: done, tests with results, risks, unverified.
