Unit P1-STORE, wave 2. Branch `feature/pg-store-v3`. Model: Sonnet 5.5, effort medium. Labels: `User Story`, `phase-1`, `wave-2`.
Files: `internal/store/`, `rpi/P1-STORE/`.
Implement on Postgres every store interface P1-CONTRACT added (see its impl-log): users, invites, auth sessions, courses, uploads, pages, claims with anchors, self-grades, disputes, course view, Mistakes. Every method takes the user id and never returns another user's rows. Tests against Postgres, including one that user A cannot read user B's course, upload, page, claim or dispute.
Done when: `go test -race ./internal/store/...` green. PR into qa with the labels above. Never merge. Report under 200 words.
