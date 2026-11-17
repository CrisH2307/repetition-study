# P3-CONTRACT plan

1. Migration 0006: users.is_instructor, entitlements, billing_events, shared_packs. DB test for checks and cascade.
2. internal/domain/v5.go: types and BillingStore, Billing, ShareStore, Sharing. User.IsInstructor.
3. api/openapi.yaml: 7 paths, /v1/me is_instructor and plan, schemas. Spec test covers v5 types and that provider ids never marshal.
4. internal/fakes/v5.go plus tests. Skip list in internal/app/handlers_test.go.
Risk: claim kind may be null in old rows; packs send it as statement.
