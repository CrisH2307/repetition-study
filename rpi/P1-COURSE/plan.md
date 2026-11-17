# P1-COURSE plan
1. `internal/course/pdf.go`: poppler via exec, 60 s text and 30 s per image timeouts, 200 page limit, bad file gives a plain error.
2. `internal/course/course.go`: `Notes` implements `domain.CourseNotes` over `CourseStore`. One block per Ready page (normalized by cite, verified_by user, verified_at now). Empty page cannot be Ready.
3. Claims: AddClaim checks the course's Ready blocks with `cite.Verify`; EditClaim checks the user's Ready blocks across courses.
4. Tests with a PDF built in the test, fakes only.
Risks: EditClaim can re-anchor to the user's other course until a GetClaim exists.
