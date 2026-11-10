# P0-INT impl log

- go.mod: pgxpool added (puddle). `go build ./...` ok.
- internal/app handlers, wiring, subjects; cmd/api; cmd/seed; Makefile seed. Builds.
- handlers_test: every served route matches api/openapi.yaml, bad input gives 400/404. Pass.
- loop_test (Postgres, throwaway schema, fake models): full loop passes. Checker-down test FAILS until P0-OUTAGE (#20) is merged: today the engine schedules even when the checker errored.
- `make migrate seed test` on a clean database: all packages ok except the checker-down test. `make seed` twice: no error, no duplicates.
- Smoke: cmd/api with STUDENT_MODEL=CHECKER_MODEL=fake served healthz, subjects, a turn over SSE, finish. Report endpoint answers 501 until the engine has GetReport.
- Merged qa with P0-OUTAGE (#36). Engine now sets `Status`, has `GetReport`, uses `ApplyTeachBackSession`. Dropped the 501 report shim (NewHandler panics if the engine lacks ReportReader) and the Status fill. Kept only the null-results guard. Finish canceled by the caller answers 499 and logs at info; deadline gives 504.
- `go build`, `go vet`, `go test -race ./...` on a fresh database: all ok, including the checker-down test.
