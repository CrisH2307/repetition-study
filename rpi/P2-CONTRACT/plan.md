# P2-CONTRACT plan

1. migrations/0005: feedback_answers, consents, import_jobs, model_spend, disputes outcome + expected_label, D43 trigger function. Tests: delete guard, v4 checks.
2. internal/domain/v4.go: types and wave 2 interfaces. internal/fakes/v4.go with tests.
3. api/openapi.yaml: 7 operations, schemas; domain JSON vs spec test. handlers_test skip list for P2-API.
Risk: a plain `SET app.delete_user` (not LOCAL) would outlive the transaction on a pooled connection. Only PrivacyStore.DeleteUser may set it, with set_config(..., true).
