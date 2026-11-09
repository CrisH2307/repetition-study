You own P0-STORE on branch feature/pg-store, from qa after P0-CONTRACT is merged.
Owner: Claude. Files: internal/store/, rpi/P0-STORE/.

Read first: docs/STATUS.md (F3), docs/DECISIONS.md, docs/ROADMAP.md.
Decisions: D24. Write plan.md, then STOP until Cris approves it (regulation 5).

Steps
1. Postgres implementation of the TeachStore interface from contract v2: writes
   teach_sessions, teach_turns, taught_claims, claim_labels; loads a concept with
   claims and sources.
2. LoadConcept returns only approved sources: the course unit, plus allowlist
   sources with reviewed_by and reviewed_at set.
3. Use a small DB interface like internal/srs and internal/planner do; do not add
   modules (go.mod belongs to P0-INT). Parameterized SQL only.
4. GetReport by session id.

Tests (Postgres, skip without DATABASE_URL, rolled-back transactions)
- Save and reload a session; restart simulation (new store instance) then Finish
  works.
- claim_labels rows exist after SaveResults; unapproved allowlist source never
  returned.
- Compile-time check that the type satisfies the domain interface.

Done when: tests pass with Postgres and with -race.

Report in under 200 words: done, tests with results, risks, unverified.
