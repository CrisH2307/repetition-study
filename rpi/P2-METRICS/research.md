# P2-METRICS research
- domain.MetricsStore gives WeekCounts (active users, sessions, checked/unchecked, disputes) and UserWeeks only.
- It has NO model cost per session and NO per-correction counts. So: cost median/p90 is not computed (unverified gap, needs a contract change from Cris). Dispute rate = disputes per checked session (not per correction).
- Upheld disputes come from EvalCaseStore.UpheldDisputes when the store has it.
- Postgres store has no MetricsStore yet (P2-STORE). cmd/export asserts the interface at run time.
