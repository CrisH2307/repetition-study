Unit P2-METRICS, wave 2. Branch `feature/metrics-export`. Model: Sonnet 5.5, medium. Labels: `User Story`, `phase-2`, `wave-2`.
Files: `internal/metrics/`, `cmd/export/`, `rpi/P2-METRICS/`.
From existing rows only (D36): active users per week, sessions per active user per week, week-4 retention of those onboarded, disputes per correction and how many were upheld, median and p90 model cost per session, unchecked rate. `cmd/export -metrics` writes CSV. No per-student names in the CSV (user ids hashed). Table tests with fixed data.
