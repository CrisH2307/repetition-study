Unit P0-TZ, wave 1, after P0-CONTRACT is on integration/p0. Branch `fix/planner-timezone`. Model: Sonnet 5.5, effort medium.
Files: `internal/planner/`, `rpi/P0-TZ/`.
Read first: `docs/improved/strategy-review-fable.md` section 4.2 (F5), `internal/planner/allocate.go` near lines 28, 154, 281, `rpi/U5/research.md`.

Problem: day boundaries use UTC, so a Toronto student gets study blocks at 04:00 to 08:00.

Steps
1. Read the user's `time_zone` (new column from P0-CONTRACT) and compute every day and week boundary in that zone.
2. Blocks fall inside configured local hours, including across a daylight-saving change (test 2026-03-08 and 2026-11-01 in America/Toronto).
3. Timed blocks stay in the library but are not the product surface (D13). Do not build the Today queue: that is Phase 1.

Done when: a test with a Toronto user shows all blocks between the configured local hours across a DST change; `go test -race ./internal/planner/...` is green.
Do not open a PR. Push the branch and stop. Report under 200 words.
