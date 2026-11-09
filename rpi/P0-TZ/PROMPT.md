You own P0-TZ on branch fix/planner-timezone, from qa. Owner: Claude. Files:
internal/planner/, rpi/P0-TZ/.

Read first: docs/STATUS.md (F5), docs/DECISIONS.md, docs/ROADMAP.md.
Decisions: D24. D13 (hide timed blocks) is Phase 1 and out of scope. Reading the
zone from users.time_zone waits for P0-CONTRACT. Write plan.md, then STOP until
Cris approves it (regulation 5).

Bug F5: internal/planner/allocate.go:28,38,281 put slots at 09:00 UTC, which is
04:00 or 05:00 in Toronto.

Steps
1. Config gets a *time.Location (default America/Toronto) and DayStartHour means
   local time. Week start and every day boundary are computed in that zone.
2. Store.Generate reads the user's zone once P0-CONTRACT adds users.time_zone.
3. Validate weekStart is a local Monday; return a clear error otherwise.

Tests
- Toronto user: every block starts between the configured local hours.
- A week containing the November DST change and one containing the March change.
- A non-Toronto zone (for example Asia/Tokyo).
- Existing determinism and hours tests stay green.

Done when: tests pass, including Postgres tests.

Report in under 200 words: done, tests with results, risks, unverified.
