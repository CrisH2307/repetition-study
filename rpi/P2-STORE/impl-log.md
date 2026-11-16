# P2-STORE impl log

- internal/store/v4.go: FeedbackStore, ImportStore, EvalCaseStore, MetricsStore, SpendLedger on Postgres. Not PrivacyStore.
- User methods scope by user (import jobs via subject owner); others get ErrNotFound. SpendLedger.Add is one upsert per UTC month.
- UpheldDisputes skips disputes with no matching claim_labels row (no checker label to compare).
- Tests: v4_store_test.go (isolation, concurrent Add on 20 connections). `go test -race -p 1 ./internal/store/...` green with Postgres.
- Unverified: behaviour on a large production dataset (metrics queries use correlated subqueries, fine at pilot size).
