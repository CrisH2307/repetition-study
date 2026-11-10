Unit P0-OUTAGE, wave 2, after P0-CITE, P0-CONTRACT and P0-SRS are on integration/p0. Branch `fix/checker-outage`. Model: Opus 5.5, effort high. Run `/code-review medium` before you report.
Files: `internal/teach/`, `rpi/P0-OUTAGE/`.
Read first: `docs/improved/strategy-review-fable.md` section 1.4 item 4, section 4.2 (F6, F7), 4.4 and 4.5, `internal/teach/engine.go` near `finalize`.

Problem: when the checker fails or times out, `finalize` marks every required claim `absent`, grades the concept Again and creates gap cards. A vendor outage silently corrupts schedules.

Steps
1. Replace the 8 second wait with one final check at Finish (synchronous, progress visible to the caller).
2. If the check fails or times out, set `Report.Status = unchecked`, write no `review_events` row and no gap cards. The report says so.
3. Calling Finish again retries the check. The session id is the idempotency key, so a successful retry writes exactly one event.
4. Do not add a per-turn strong-model check. Per-turn matching is Phase 1.

Done when: a test with a failing checker returns `unchecked` and `review_events` gains no row; a retry after the checker recovers writes one event and gap cards; `go test -race ./internal/teach/...` is green.
Do not open a PR. Push the branch and stop. Report under 200 words.
