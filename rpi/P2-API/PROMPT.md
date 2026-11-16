Unit P2-API, wave 3. Branch `feature/api-v4`. Model: Sonnet 5.5, high. Labels: `User Story`, `phase-2`, `wave-3`.
Files: `internal/app/`, `cmd/api/`, `cmd/export/`, `rpi/P2-API/`.
Wire every v4 endpoint behind sign-in, remove them from the test skip list, add `cmd/export -deletions`. Isolation test: user A gets 404 on user B's export, feedback and imports. Full loop with the fake model still green.
