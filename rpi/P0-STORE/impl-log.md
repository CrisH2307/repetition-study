# P0-STORE impl log
- store.go written. `go build ./...` ok.
- store_test.go: `go test -race ./internal/store/...` passes (6 tests) against dev Postgres; skips cleanly without DATABASE_URL.
