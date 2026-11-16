# P1-STORE impl log

- AuthStore, CourseStore, ReviewStore on `*store.Store`. `go test -race ./internal/store/...` green against Postgres 14 (11 tests, 0 skipped).
- `TestUserCannotReadAnotherUsersRows`: user B gets ErrNotFound from all 30 methods on A's course, concept, upload, page, claim, session, dispute.

## Choices to know (P1-COURSE, P1-API)

- Blocks of a page get ordinal `page_number*100000 + index` (UNIQUE source_id, ordinal). The ordinal the caller passes is ignored.
- SetPageReady creates the upload's source on first use (kind course, `subjects` link, reviewed_by = user id) and rebuilds `sources.text` from the page blocks (joined by a blank line). CourseSources returns course sources of the course that have text.
- SetPageReady/NotReady move an upload between `ready` (all pages Ready) and `needs_checking`. They never touch a `reading` upload.
- A live claim anchored to a page refuses both un-ready and re-ready (UserError). Deleted claims lose their anchor so the blocks can go.
- Create/UpdateClaim check only that the block is on a Ready page of the claim's course and set source_id and locator ("p.N"). Quote check stays in internal/cite.
- SavePages overwrites an unchecked page with the same number (re-read) and keeps a Ready one.
- Mistakes: only labels with a citation; page is found by searching the cited source's blocks for the quote.
- CreateCourse keeps weight as sent (0 stays 0).
- SessionOwner does not hide archived courses.
