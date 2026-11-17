# P3-SHARE impl log

- store/share.go and TestShareStore: green on a local Postgres 14 with migrations 0001 to 0006.
- internal/share and tests (no quote or notes in preview, import anchors to receiver notes, needs_quote, instructor copy, revoked and foreign 404): `go test -race` green.
- cmd/instructor and test: green. Full `go test -race ./...` green.
