You own P0-CONTRACT on branch feature/contract-v2, from qa. Owner: Cris (Claude
drafts under Cris). Files: internal/domain/, api/, migrations/, internal/fakes/,
rpi/P0-CONTRACT/. This is the only Phase 0 unit allowed to change the contract.

Read first: docs/STATUS.md (contract gaps, F6, F7, F14), docs/DECISIONS.md, docs/ROADMAP.md.
Decisions: D23 (report status unchecked), D24. Write plan.md with the exact type,
interface, OpenAPI and SQL diff, then STOP until Cris approves it (regulation 5).

Scope (contract v2, additive where possible)
1. Pack loader type for {subject, sources, concepts} as used by content/unit.json.
2. TeachStore interface in domain (move or mirror internal/teach/store.go methods)
   and TeachEngine.GetReport(sessionID).
3. Explicit event id: Scheduler.ApplyTeachBack, ReviewCard and Override take an
   event id parameter (session id at Finish). Keep srs.WithEventID working until
   P0-SRS switches. To keep the build green without editing srs, teach or
   planner (outside this boundary), add the event-id methods beside the old ones
   and mark the old ones deprecated; P0-SRS and P0-OUTAGE remove them.
4. DueItem.Reason.
5. [D23] Report.Status with values checked and unchecked; ClaimResult unchanged.
6. users.time_zone (IANA name, default America/Toronto) in a new migration
   0003_*.sql. Never edit 0001 or 0002.
7. Fakes updated; fix fakes.Scheduler.ReviewCard hardcoding "concept-1" (F14).
8. OpenAPI updated to match; validate with a real validator (for example
   `npx @redocly/cli lint`, verify the current command first).
Out of scope: claim_states, courses, enrollments, source_blocks (Phase 1).

Tests
- go build ./... and go test ./... green (unit packages may need adapter shims;
  list every compile break and fix only fakes; report others to Cris).
- Migration test: 0003 applies on a database with 0001 and 0002 and is idempotent.

Done when: build and tests green, validator clean, a list of every unit that must
adapt is in impl-log.md.

Report in under 200 words: done, tests with results, risks, unverified.
