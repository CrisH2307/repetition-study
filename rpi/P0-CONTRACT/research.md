# P0-CONTRACT research

- OpenAPI validator: `github.com/getkin/kin-openapi` v0.152.0 (released 2026-10-22; v0.153.0 is two days old, skipped).
  Verified in module source: `openapi3.NewLoader()`, `(*Loader).LoadFromFile(path) (*T, error)`,
  `(*T).Validate(ctx, ...ValidationOption) error`, `(*T).ValidateSchemaJSON(*Schema, any, ...)`,
  `T.Components.Schemas map[string]*SchemaRef`.
- `internal/teach/engine.go:15` asserts `domain.TeachEngine` and `internal/srs/scheduler.go:16` asserts
  `domain.Scheduler`. Adding a method to either interface breaks those packages, which this unit cannot edit.
  So `GetReport` and the session-keyed teach-back live in new interfaces (`ReportReader`, `SessionScheduler`).
- `internal/teach` tests import `internal/fakes`, so fakes must never import teach (test import cycle).
  The `teach.Store` == `domain.TeachStore` check lives in `internal/domain/contract_test.go` instead.
- `content/unit.json` has `subject`, `sources`, `concepts[].claims`. `Pack` keeps those JSON keys.
- `ALTER TABLE ... ADD COLUMN` does not fire row triggers, so 0003 is safe with the append-only trigger.
- A dir with only `_test.go` files passes `go build ./...` and `go vet ./...` (checked on Go 1.26.3).
- Dev Postgres on :5432 is shared with other worktrees, so tests run against fresh scratch databases.
