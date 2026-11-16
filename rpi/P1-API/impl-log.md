# P1-API impl log

- Handlers, wiring, cap, export written. `go build ./...` green.
- `TestHandlersMatchOpenAPI` now walks every v3 operation over the fakes. `notServed` keeps only the four Phase 0 operations the spec lists but nobody serves. Green.
- Phase 0 loop tests sign in as the demo user. They failed once: seeded sources have no `subject_id`, so the checker saw no source and downgraded a citation. Fixed in `userTeach`. Green.
- `TestIsolation` (Postgres): 27 requests as the other student all 404, all 401 without a sign-in, lists hold only the caller's rows. Green.
- `TestFullLoopV3`, `TestSpendCapGivesUncheckedReport`, `TestCapLLMResetsEachMonth`, `TestUploadRefusals`. Green.
- `web/e2e/loop.e2e.ts`: 2 passed against the Go API. Mock config 6 passed.
- `go test -race -p 1 ./...` green. Without `-p 1` the `migrations` test can fail when run alongside other packages on the shared dev database.
