# U0 impl-log

- Scaffold: go.mod with go-fsrs/v4, pgx/v5, genai, anthropic-sdk-go pinned via internal/deps; compose.yaml, Makefile, .env.example, empty rpi files. `go mod tidy` clean.
- internal/domain types and interfaces. `go vet` clean.
- internal/fakes, deterministic. `go test ./internal/...` ok (determinism, wrong explanation pulls date earlier).
- migrations 0001 core, 0002 append-only trigger, cmd/migrate, scripts/seed.sql. `make migrate` and `make seed` are idempotent (ran twice).
- Trigger test: passes, and fails with the update/delete trigger dropped (mutation check).
- api/openapi.yaml (15 paths, 24 schemas, no broken $refs, linted with a local script only) and cmd/api /healthz. curl returned ok.
- Full `make up migrate test` green. PRs are split into 4 branches, see plan.md.
