You own P0-OUTAGE on branch fix/checker-outage, from qa after P0-CITE and
P0-CONTRACT are merged. Owner: Claude. Files: internal/teach/, rpi/P0-OUTAGE/.

Read first: docs/STATUS.md (F6, F7), docs/DECISIONS.md, docs/ROADMAP.md.
Decisions: D23 (unchecked behaviour) and D5 (final check at Finish), D24. If D23
is Pending, stop after research.md. If D5 is Pending, do steps 1 to 3 only.
Write plan.md, then STOP until Cris approves it (regulation 5).

Bug F6: if every check fails, finalize (internal/teach/engine.go:311-316) marks
all required claims absent, which grades Again and creates gap cards.
Bug F7: Finish calls ApplyTeachBack without an event id (engine.go:267).

Steps
1. If no check completed, or the latest check failed, the report status is
   unchecked, no ApplyTeachBack call happens, and no review_events row is written.
2. Pass the session id as the event id to ApplyTeachBack (contract v2).
3. A second Finish after a restart returns the stored report and writes nothing.
4. [D5] Finish runs one final full check, synchronous with a timeout, instead of
   relying on the last async per-turn check.

Tests (fakes)
- Checker always fails: report unchecked, scheduler fake records zero calls.
- Checker times out at Finish: same.
- Finish twice with the same session: one scheduler call.
- Existing teach tests stay green; go test -race ./internal/teach.

Done when: tests above pass.

Report in under 200 words: done, tests with results, risks, unverified.
