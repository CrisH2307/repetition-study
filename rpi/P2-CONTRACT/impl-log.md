# P2-CONTRACT impl log

- 0005 migration + 2 DB tests (D43 guard, v4 checks). Applied on Phase 1 copy; race tests green.
- domain v4 types, interfaces, fakes + 4 fake tests. Green.
- OpenAPI v4 paths and schemas; spec test covers every v4 type; v4 ops on handlers_test skip list. Full `go test -race ./...` green with Postgres.

## Interfaces for wave 2 (internal/domain/v4.go)
- FeedbackStore (SaveFeedback, GetFeedback, SaveConsent, Consents): P2-WEB2/P2-API, store by P2-API.
- PrivacyStore (ExportUser, DeleteUser): P2-PRIVACY. DeleteUser is the only code that sets app.delete_user, via set_config(..., true) in one tx.
- ImportStore + Importer: P2-IMPORT. Importer must add claims through CourseNotes.AddClaim (D38).
- EvalCaseStore (ResolveDispute, UpheldDisputes): P2-EVAL2. Dev split only (D39).
- MetricsStore (WeekCounts, UserWeeks): P2-METRICS. Read-only.
- SpendLedger (Add, MonthTotal) over table model_spend: P2-API replaces the in-memory count in capLLM.
