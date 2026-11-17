# P3-SHARE research

- Contract: internal/domain/v5.go (ShareStore, Sharing), migration 0006. No users.email column, so cmd/instructor takes `-user <id>`, not `-email`.
- Quote for each imported claim: best sentence of the receiver's Ready blocks sharing at least half of the claim's words (4+ letters), cut from the block, then checked by cite.Verify and again by CourseNotes.AddClaim (D28). No model.
- Claims with no quote are listed in needs_quote and not saved (contract v5).
- Instructor notes are copied as a new upload per owner upload, then ReadUpload and MarkReady in the receiver's course (their own page check). Page breaks are lost (locator p.1).
- Owner name and flag come from store.UserByID (new, not in the contract). SessionUser still does not select is_instructor: P3-API.
