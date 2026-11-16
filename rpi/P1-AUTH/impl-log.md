# P1-AUTH impl log

- internal/auth service, cookie, middleware, CSRF. `go test -race ./internal/auth` green.
- cmd/invite prints the link. `go test -race ./cmd/invite` green, `go build ./...` green.
