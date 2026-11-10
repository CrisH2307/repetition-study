# P0-SRS research
- Contract v2 (migrations/0003): `review_events.session_id` + unique partial index `review_events_teach_back_session_uniq` (one teach_back per session). `domain.SessionScheduler.ApplyTeachBackSession` is the new interface.
- Existing `run` already holds a per-concept advisory lock, so concurrent retries queue; the index is the backstop.
- Dev DB (localhost:5432) is only migrated to 0002, so tests build their own throwaway DB with all migrations.
- Benchmark `BenchmarkSchedule1000`: 1.6 ms per 1,000 concepts (limit 20 ms).
