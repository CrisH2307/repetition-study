# P3-API impl log
- v5.go, Wire, SessionUser, notServed: `go test -race ./...` green on local Postgres 16 with migrations 0001 to 0007.
- TestIsolationV5: other student 404, revoked 404, signed out 401, instructor flag on /v1/me.
- Playwright: real config 3 passed (new share and import test), mock config 19 passed.
