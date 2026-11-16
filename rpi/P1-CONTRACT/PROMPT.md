Unit P1-CONTRACT, wave 1. Branch `feature/contract-v3`. Model: Opus 5.5, effort high. Labels: `User Story`, `phase-1`, `wave-1`.
Files: `internal/domain/`, `api/`, `migrations/`, `internal/fakes/`, `rpi/P1-CONTRACT/`.
Read first: `docs/PLAN.md` (Phase 1), `docs/DECISIONS.md` D26 to D34, `docs/design/ui-design.md`, current `internal/domain`, `api/openapi.yaml`, `migrations/`.

Contract v3. Additive migration `0004_contract_v3.sql` (fix forward, no data loss: Phase 0 rows stay with the default user).
1. Users and sign-in (D29): invites (token hash, created, used_at, user_id) and auth sessions (token hash, user_id, expires_at). Scope courses (subjects) by user; sources, uploads and claims reach the user through their course.
2. Uploads and pages (D27): upload (course, filename, pdf bytes or pasted text, status reading/needs_checking/ready, error), page (upload, number, raw text, edited text, status needs_checking/ready, image png). A Ready page becomes source blocks of an approved course source.
3. Claim checklist (D28): claims gain kind (definition, statement, condition, method), author user, and an anchor (source block id, quote). Keep existing claims valid.
4. Self-grades per session and claim (right, not_sure, wrong) and disputes (session, claim, user, note, created_at, status open/reviewed) (D30). Neither touches review_events.
5. Today and week (D31): types for a Today item (concept, course, kind teach/recall, minutes, due_at, reason) and a week plan (per course hours used/budget with reason, per day items, notes).
6. Course view: notes with status, study cards with recall and explanation state, a Mistakes list (claim, label, quote, page).
7. Domain interfaces for the stores, course logic, auth and Today that wave 2 implements; fakes for each.
8. OpenAPI: sign-in by invite, sign-out, me; courses CRUD; uploads, pages (get, edit, mark ready, image); claims CRUD per concept; self-grade; dispute; today; week; course view. Errors in plain words. Keep existing endpoints working.

Done when: `go build ./... && go vet ./... && go test -race ./...` green with Postgres, migration applies on a copy of a Phase 0 database, OpenAPI validated by the existing test. Write the endpoint list in `rpi/P1-CONTRACT/impl-log.md` for wave 2 and 3.
Push, open a PR into qa (`gh pr create --base qa --label "User Story" --label phase-1 --label wave-1`, body "Closes #<issue>", tests, unverified). Never merge. Report under 200 words.
