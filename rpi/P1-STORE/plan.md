# P1-STORE plan

Files: internal/store/{common,auth,course,review}.go, v3_test.go.
1. AuthStore. 2. CourseStore (courses, concepts, uploads, pages, blocks, claims). 3. ReviewStore (grades, disputes, view, state).
Every query joins up to subjects.user_id and hides archived courses and deleted claims. Bad uuids give ErrNotFound.
Tests: Postgres schema per test; one test makes user B call every method against user A's ids.
Risks: ordinal scheme for page blocks; upload status auto-moves with page status.
