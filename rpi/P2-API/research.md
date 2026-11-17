# P2-API research
- Store already implements FeedbackStore, ImportStore, PrivacyStore, SpendLedger, MetricsStore (internal/store). Importer is internal/importer.
- Weekly question: no source in the repo, so a short rotation by code in internal/app/v4.go (first is the G2 question, D40). Unverified wording; Cris may edit.
- Cookie clear on delete is set in app (auth.HTTP.clear is private and internal/auth is outside this unit).
