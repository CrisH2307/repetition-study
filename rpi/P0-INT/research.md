# P0-INT research

Verified in the code (not assumed):
- `store.Store` implements `TeachStore` and `PackLoader` over a small `DB` interface. It says P0-INT adds pgxpool to go.mod. Done: `go get .../pgxpool` added only `puddle/v2` (indirect). `go mod tidy` also marks `invopop/jsonschema` direct (internal/llm imports it).
- `srs.Scheduler` has `ApplyTeachBackSession`, but `teach.Engine.Finish` still calls `ApplyTeachBack` (P0-OUTAGE changes this). The engine caches the finished report in memory, so a repeat Finish in one process returns the same event.
- `teach.Engine` has no `GetReport` and no `unchecked` path yet (P0-OUTAGE, issue #20). It never sets `Report.Status`.
- `srs.Scheduler.Due` returns items with an empty `Reason`: `ReasonedDue.Reason` shadows `DueItem.Reason`. `DueWithReasons` has the text. Bug is in internal/srs, not patched here.
- Planner hours: demand is `ceil(weekly_hours * min(1, gap/0.5))` with gap = target 0.9 minus mean projected recall at the exam date. With the fixture's two concepts the budget only moves when the baseline recall is high, so the loop test puts the clock at Monday 2026-12-07 (exam 2026-12-15) after two good teach-backs.
- kin-openapi v0.152.0: `Schema.VisitJSON(v, EnableFormatValidation())` validates a response body against the spec schema without a router (gorilla/mux is not in go.mod). I did not upgrade it.
- Only one Postgres can own port 5432. `make up` from this worktree failed (another project holds the port), so the clean-DB run used a new database on the running server.

Unverified: real Gemini and Claude behaviour (no keys used), `make up` on a machine with a free port 5432.
