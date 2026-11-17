# P2-IMPORT impl log

- internal/importer: Importer over ImportStore, CourseStore, ReviewStore (concept lookup by name) and CourseNotes. Finds the block holding each quote with cite.Verify, then AddClaim does the real check.
- Plain-words row errors never repeat cell text; all error text is prefixed with ' if it starts with = + - @ tab CR (formula guard). 500 row limit, header check, BOM ok.
- 4 tests with fakes, `go test -race ./internal/importer` green.
- Unverified: Required is set true for every imported claim; no real Postgres run; not wired to an API route (P2-API).
