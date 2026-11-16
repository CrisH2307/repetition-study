Unit P2-PRIVACY, wave 2. Branch `feature/privacy-export`. Model: Sonnet 5.5, high. Labels: `User Story`, `phase-2`, `wave-2`.
Files: `internal/privacy/`, `internal/store/privacy.go`, `internal/store/privacy_test.go`, `rpi/P2-PRIVACY/`.
D37: export everything the signed-in user owns (courses, uploads text, pages, claims, sessions, turns, reports, self-grades, disputes, review events, feedback) as one JSON; never another user's rows (test). Delete request: record it once, a store method lists open requests (P2-API adds `cmd/export -deletions`). No row is deleted.
