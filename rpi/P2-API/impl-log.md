# P2-API impl log
- v4 handlers and wiring; removed from notServed. `go test ./internal/app` green.
- Ledger-backed cap (spendRec, metered, capLLM). Restart and Postgres tests green.
- Cost per checked session in `cmd/export -metrics`. Tests green.
- Full `go test -race ./...` with Postgres green. Playwright e2e (2 tests, fresh seeded DB) green.
