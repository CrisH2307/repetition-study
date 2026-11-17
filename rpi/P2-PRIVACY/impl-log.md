# P2-PRIVACY impl log
- privacy.go: ExportUser, DeleteUser. `go vet` clean.
- privacy_test.go: export has no other user's rows, delete scopes by table counts, bob export identical after, rollback on a failing step keeps review events, re-delete is ErrNotFound, setting does not leak, review_events delete outside the path fails. Green with -race; `go test` of every package except migrations is green.
- migrations package fails on the shared dev DB because it has not had 0005 applied (not this unit).
