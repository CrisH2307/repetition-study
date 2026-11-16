# P0-OUTAGE impl log

- Research and plan written. No tests.
- Finish runs one synchronous final check; outage gives an unchecked report with no save, no event, no gap cards; retry keyed by session id; per-turn checks and FinishWait removed; `New` takes `domain.SessionScheduler`; Engine implements `ReportReader`. `go test -race ./internal/teach/...` ok.
- /code-review medium: unchecked report now returns `results: []` (API requires an array); failed check logged once. Tests ok. Skipped, listed as risks: retry after a post-commit error can re-save different labels; equal caller and check deadlines count as caller cancel; no worker retries unchecked sessions; `pickProbe` hint branches are unused until the Phase 1 matcher.
