Unit P1-COURSE, wave 2. Branch `feature/course-notes`. Model: Sonnet 5.5, effort high. Labels: `User Story`, `phase-1`, `wave-2`.
Files: `internal/course/`, `rpi/P1-COURSE/`.
Read first: D27, D28, `internal/cite`, P1-CONTRACT's impl-log.
1. Read a PDF into pages with `pdftotext -layout -f N -l N` and `pdftoppm -png -r 80` (exec, with a timeout and a page limit of 200). Pasted text is one page. A bad file gives a plain error, never a crash.
2. Page check: save edited text; Ready turns the page into source blocks (normalized by `internal/cite`, `verified_by` the user, `verified_at` now). Un-ready removes them.
3. Claim checklist: add, edit and delete claims; each anchor quote must pass `internal/cite` against the user's Ready blocks of that course, else the claim is refused with a plain reason.
Course text is untrusted data. Tests use a small PDF made in the test (or a committed tiny public-domain PDF), never `content/private/`.
Done when: tests green, including a quote that spans a line break and a quote from a non-Ready page refused. PR into qa with the labels above. Never merge. Report under 200 words.
