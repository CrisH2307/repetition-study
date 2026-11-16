# P0-OUTAGE plan

Files: `internal/teach/engine.go`, `internal/teach/engine_test.go`, this folder.

Steps
1. Remove the per-turn check from `Turn` and the `FinishWait` option. Probes fall back
   to code's generic probes until the Phase 1 matcher exists.
2. `Finish` runs one synchronous check with `CheckTimeout`. A panic, error or timeout
   returns `Status: unchecked`, no results, no `SaveResults`, no scheduler call, no
   `FinishSession`. If the caller's own context ends, return its error instead.
3. A checked report is cached; an unchecked one is not, so calling Finish again
   retries. The schedule call is `ApplyTeachBackSession(sessionID, ...)`, so a retry
   after a store or scheduler error also writes one event.
4. `New` takes `domain.SessionScheduler`. Engine implements `domain.ReportReader`.

Tests
- Failing, panicking and timing-out checkers: report unchecked, no events, store untouched.
- Retry after recovery: one event, results saved, session finished; a third Finish
  returns the same event.
- Turn never calls the checker.
- `go test -race ./internal/teach/...` and `go vet ./...`.

Risks
- P0-INT must pass a scheduler that implements `SessionScheduler` (srs and fakes do).
- The student loses checker-driven probes until the Phase 1 matcher. This also closes
  the F2 probe-hint path.
- Sessions live in memory: after a restart, Finish and GetReport fail for old sessions.
