Unit P1-API, wave 3. Branch `feature/api-v3`. Model: Sonnet 5.5, effort high. Labels: `User Story`, `phase-1`, `wave-3`.
Files: `internal/app/`, `cmd/api/`, `cmd/export/`, `Makefile`, `go.mod`, `go.sum`, `rpi/P1-API/`.
Wire every v3 endpoint in `api/openapi.yaml` to the wave 2 code, behind the auth middleware (healthz and invite redemption stay open). Uploads read in the background and the page list shows the status. Teach uses only the signed-in user's Ready blocks as sources. `cmd/export -disputes` writes a CSV for Dino. Monthly spend cap from the environment (D33): when reached, checks report `unchecked`.
Tests: a contract test that every OpenAPI path has a handler; an isolation test that user A gets 404 for each of user B's ids on every endpoint; the full loop with the fake model from upload to report.
Done when: `go test -race ./...` green. PR into qa with the labels above. Never merge. Report under 200 words.
