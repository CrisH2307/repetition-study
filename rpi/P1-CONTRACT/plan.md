# P1-CONTRACT plan

1. `migrations/0004_contract_v3.sql`: invites, auth_sessions, uploads, pages, self_grades, disputes; new nullable columns on subjects, sources, source_blocks, claims. Test the checks; apply on a copy of the Phase 0 DB.
2. `internal/domain/v3.go`: types for users, uploads, pages, claim checklist, self-grades, disputes, Today, week, course view; `ErrNotFound`, `UserError`; interfaces AuthStore, Auth, CourseStore, ReviewStore, CourseNotes, TodayPlanner. Claim gains Kind, AuthorUserID, Anchor.
3. `internal/fakes/v3.go`: one in-memory fake per interface, user-scoped, with tests (auth flow, course loop, isolation, Today).
4. `api/openapi.yaml` 0.3.0: cookie auth and every v3 path; extend the spec-drift test.

Risks: the `internal/app` spec-coverage test needs the v3 paths on its skip list until P1-API (outside this boundary).
