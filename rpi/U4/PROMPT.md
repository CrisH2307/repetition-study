You own U4 on branch feature/scheduler. Files: internal/srs/, rpi/U4/.

Build Scheduler on go-fsrs with default FSRS-6 parameters. Install the latest
stable module, then read its actual API and record the version in
rpi/U4/research.md. Use the Clock interface everywhere.

Model
- A concept has one schedule. Cards under it have their own FSRS state.
- ApplyTeachBack takes the worst result across required claims:
  contradicted or absent gives Again, partial gives Hard, all supported gives Good.
  It never gives Easy.
- Every state change appends a review_events row with due and stability before
  and after. Teach-back rows use event_type teach_back, so this mapping can be
  analyzed separately later.
- Writes are idempotent, using a client-supplied event id.
- Override changes the due date only, leaves memory state alone, and logs a
  manual_override event with a reason.
- For each contradicted, partial or absent result, create one editable card with
  the citation attached. Do not create duplicates of an existing open gap card.
- Do not show two cards from the same concept in a row when others are due.
- Due returns concepts and cards with the reason each is due.

Tests
- All three grade mappings, using a fixed clock.
- A concept due in 7 days has an earlier due date after a contradicted teach-back.
- Override leaves stability unchanged.
- A retried event writes once.
- 1,000 concepts schedule in under 20 ms, measured with a Go benchmark.

Report in under 200 words.
