# U4 plan

1. `logic.go`: pure grade mapping, FSRS step, interleave. Tests + 1,000 concept benchmark.
2. `scheduler.go`: Postgres store. One tx per write with a per-concept advisory lock; event id checked first (idempotent), then concept_schedules upsert, then review_events insert.
3. Gap cards from contradicted, partial, absent results, deduped by claim.
4. `Due` / `DueWithReasons` with round-robin interleave across concepts.
5. DB tests (skip without DATABASE_URL): mappings, 7-day pull earlier, override, retry, cards, due.

Risks: results with no claim id count toward the grade; same-instant events tie when picking the latest event for a due reason; test rows stay in the dev DB (append-only table).
