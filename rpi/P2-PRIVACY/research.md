# P2-PRIVACY research
- Ownership: every table reaches its user through subjects.user_id, or carries user_id (self_grades, disputes, feedback_answers, consents, digests, auth_sessions, invites). Sources are the user's only when sources.subject_id is one of theirs; allowlist/shared sources (subject_id NULL) are never deleted.
- Trigger from 0005 allows a review_events delete only for rows whose concept belongs to the user in `app.delete_user`. So DeleteUser also scopes the DELETE itself; the trigger is the second lock.
- store.DB has only Begin, so the export snapshot uses `SET TRANSACTION ISOLATION LEVEL REPEATABLE READ, READ ONLY` as the first statement.
- No `internal/privacy/` package was needed: the interface is in domain and the Postgres code is in store. The API confirm step ("DELETE" or course name) belongs to P2-API.
