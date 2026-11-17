# P3-CONTRACT plan

1. Migration 0006: users.is_instructor, shared_packs (billing dropped by Cris). DB test for checks and cascade.
2. internal/domain/v5.go: ShareStore, Sharing and types. User.IsInstructor.
3. api/openapi.yaml: 4 paths, /v1/me is_instructor, schemas. Spec test covers v5 types and that the share owner never marshals.
4. internal/fakes/v5.go plus tests. Skip list in internal/app/handlers_test.go.
Risk: claim kind may be null in old rows; packs send it as statement.
