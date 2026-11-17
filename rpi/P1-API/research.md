# P1-API research

- Go 1.25 net/http: `http.NewCrossOriginProtection` is already used by `auth.Protect`. Method and path patterns in `ServeMux` give `r.PathValue`.
- `auth.HTTP.Require` puts the user in the context. Open routes: `/healthz`, `/v1/auth/redeem`. Sign-out sits behind `Require`.
- The scheduler (`internal/srs`) and `planner.Store.FurtherPlanning` are not user aware. Ownership is checked in the handlers (`Owner`), the due list is filtered to the user's concepts, and the planner is built per user (`PlannerFor`).
- `teach.Store.LoadConcept` has no user. `userTeach` reads the user from the request context, takes the user's live required claims and the course sources.
- `cmd/seed` (`LoadPack`) leaves `sources.subject_id` empty, so `CourseSources` is empty for seeded packs. `userTeach` also takes the sources the claims cite. Unverified fix on the store side: set `subject_id` in `LoadPack`.
- Chromium accepts a `Secure` cookie on `http://localhost`, so the Playwright run needs no `COOKIE_SECURE=false`.
- Unverified: real Poppler PDF upload end to end (tests paste text; the fake notes read PDFs as "fake page").
