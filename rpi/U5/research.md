# U5 research
- Contract: domain.Planner {Generate(ctx,userID,weekStart), FurtherPlanning(ctx,weekStart)}; domain.Clock. No edits to domain.
- FSRS: go-fsrs v4.0.0 `Parameters.ForgettingCurve(elapsedDays, stability)` is public; R at exam date = that with elapsed = exam - last_review. New card or no last review = 0.
- Schema: concept_schedules (one FSRS state per concept: stability, state int (0=New), last_review_at); subjects.user_id; plan_blocks has locked; review_events append-only (cannot delete test rows, so DB tests run in a rolled-back tx).
- FurtherPlanning has no userID: Store uses domain.DemoUserID by default (field UserID).
- Unverified: exact hours semantics. Prompt says respect weekly_hours AND worse retention raises hours. Read as: weekly_hours is a ceiling, hours demanded = ceiling * min(1, gap/FullGap). Proposal, in Config.
- Contract gap: ConceptProgress has one Recall field and SubjectProgress has no change reasons or gaps. Richer data is returned by Store.Progress; FurtherPlanning maps what fits. Cris/INT decides if domain grows.
