# P3-SHARE plan

1. store/share.go: CreateShare, RevokeShare, ShareByToken, SetInstructor, UserByID. Test on Postgres.
2. internal/share: Create, Revoke, Preview, Import over CourseStore, ReviewStore, CourseNotes. Tests on fakes plus real course.Notes.
3. cmd/instructor: -user, -off.
Risks: word-overlap quote matching misses paraphrases (they land in needs_quote); no list of my links yet.
