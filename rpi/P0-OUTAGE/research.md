# P0-OUTAGE research

Checked on qa at 9c8f14c.

- `internal/teach/engine.go`: `Turn` starts `runCheck` per turn; `Finish` waits up to
  `FinishWait` (8 s), then `finalize` marks every unmentioned required claim `absent`.
  A failed or slow checker therefore grades Again and makes gap cards (F6).
- Gap cards are made inside the scheduler (`internal/srs/scheduler.go` `applyTeachBack`),
  so not calling the scheduler means no `review_events` row and no gap cards.
- Contract (P0-CONTRACT): `Report.Status` with `ReportChecked` / `ReportUnchecked`;
  `ReportReader.GetReport`; `SessionScheduler.ApplyTeachBackSession` (idempotent per
  session). Both `srs.Scheduler` and `fakes.Scheduler` implement it.
- The engine only calls `ApplyTeachBack` on its scheduler, so `New` can take a
  `domain.SessionScheduler` directly. Callers passing `*srs.Scheduler` still compile.
  `internal/app` does not exist on qa yet (P0-INT builds it).
- DECISIONS D5 (approved): one final check at Finish, plus a cheap per-turn matcher
  that never sees source prose. The matcher is Phase 1, so per-turn checks go away now.
- Strategy 4.4: final check is synchronous with a progress indicator. The domain
  `Finish` signature returns only a `Report`, so progress display belongs to the
  handler and UI. Unverified: how P0-INT shows it.
- Store `FinishSession` sets status finished. An unchecked session is not finished.
