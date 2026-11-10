# U5 plan
1. allocate.go: Config (all constants), pure Score + Allocate (greedy 30 min blocks, slot placement, locked blocks, teach-before-review, teach >= 24h ahead, reasons). Tests: determinism, hours cap, locked kept, worse retention raises hours, teach order/lead, no-concept subject.
2. store.go: pgx loader, Generate (replace unlocked blocks of the week in one tx). Tests in a rolled-back tx.
3. progress.go: Progress (recall, explanation, gaps with citations, next week hours and reasons) and FurtherPlanning mapping. Tests.
Risks: hours semantics (see research), UTC day windows, current-week lead time.
