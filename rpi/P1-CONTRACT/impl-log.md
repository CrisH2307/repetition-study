# P1-CONTRACT impl log

- 0004 migration added. Applied on `p1_contract_copy` (pg_dump of the Phase 0 dev DB): row counts unchanged, 49/49 sources got a course, rerun is a no-op. `TestContractV3Checks` green.
- `internal/domain/v3.go` types and interfaces. `go build ./...` green.
- `internal/fakes/v3.go` with tests. `go test -race ./internal/fakes` green.
- OpenAPI 0.3.0 with v3 paths. `go test ./api` green (spec valid, 15 v3 types checked against it).

## For wave 2 and 3

Rules every implementation follows:
- Every store method takes the signed-in user id. Rows of another user give `domain.ErrNotFound` (HTTP 404), same as missing rows.
- `*domain.UserError` is a plain-words refusal (HTTP 400). Its `Msg` is shown as is.
- Return empty slices, never nil: the spec's arrays are not nullable.
- Courses are archived (`subjects.archived_at`), claims soft-deleted (`claims.deleted_at`). Never delete rows that `review_events` can point at.
- Self-grades and disputes never touch `review_events` or a schedule (D30).
- Blocks from a Ready page carry `source_blocks.page_id`. Un-ready deletes them, refused while a live claim anchors to one.
- `domain.TeachStore.LoadConcept` is not user-scoped (pinned to `teach.Store`). P1-API checks ownership with `CourseStore.GetConcept` first.

Interfaces (`internal/domain/v3.go`), fakes in `internal/fakes/v3.go`:

| Interface | Implements | Fake |
| --- | --- | --- |
| `AuthStore` | P1-STORE | `fakes.AuthStore` |
| `CourseStore` | P1-STORE | `fakes.CourseStore` |
| `ReviewStore` | P1-STORE | `fakes.ReviewStore` |
| `Auth` | P1-AUTH | `fakes.Auth` |
| `CourseNotes` | P1-COURSE | `fakes.CourseNotes` |
| `TodayPlanner` | P1-TODAY | `fakes.TodayPlanner` |

New tables: `invites`, `auth_sessions`, `uploads`, `pages`, `self_grades`, `disputes`.
New columns: `subjects.archived_at`, `sources.subject_id`, `source_blocks.page_id`, `claims.kind/author_user_id/anchor_block_id/anchor_quote/created_at/deleted_at`.

Endpoints (all need the `session` cookie except redeem; P1-API wires them):

| Method and path | Calls |
| --- | --- |
| POST /v1/auth/redeem | Auth.Redeem, sets cookie |
| POST /v1/auth/sign-out | Auth.SignOut, clears cookie |
| GET /v1/me | Auth.UserFor |
| GET, POST /v1/courses | CourseStore.ListCourses, CreateCourse |
| GET, PATCH, DELETE /v1/courses/{id} | GetCourse, UpdateCourse, ArchiveCourse |
| GET /v1/courses/{id}/view | ReviewStore.CourseView |
| POST /v1/courses/{id}/concepts | CourseStore.CreateConcept |
| GET /v1/courses/{id}/blocks | CourseStore.ReadyBlocks |
| GET, POST /v1/courses/{id}/uploads | ListUploads; CreateUpload then CourseNotes.ReadUpload in the background (202) |
| GET /v1/uploads/{id} | GetUpload |
| GET /v1/uploads/{id}/pages | ListPages |
| GET, PATCH /v1/pages/{id} | GetPage, CourseNotes.EditPage |
| GET /v1/pages/{id}/image | PageImage (image/png) |
| POST /v1/pages/{id}/ready, /unready | CourseNotes.MarkReady, MarkNotReady |
| GET, POST /v1/concepts/{id}/claims | ListClaims, CourseNotes.AddClaim |
| PATCH, DELETE /v1/claims/{id} | CourseNotes.EditClaim, DeleteClaim |
| POST /v1/teach/sessions/{id}/self-grades | ReviewStore.SaveSelfGrades (204) |
| POST /v1/teach/sessions/{id}/disputes | ReviewStore.CreateDispute (201) |
| GET /v1/today | ReviewStore.StudyState then TodayPlanner.Today |
| GET /v1/week?week= | ReviewStore.StudyState then TodayPlanner.Week |

Phase 0 endpoints are unchanged and keep working; after P1-API they act as the signed-in user.
`cmd/export -disputes` uses `ReviewStore.AllDisputes`.
