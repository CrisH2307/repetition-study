# Research
- Baseline (laptop, 500 users): queries are sub-millisecond on this data (EXPLAIN ANALYZE), no missing index. Cost is round trips: Postgres in docker used ~150% CPU at 500 users, the API little.
- Round trips per request: today and week 5 (session + 4 in StudyState); course_view 9 (session, course, course again inside ListUploads, uploads, cards, one claims query per card, mistakes).
- Pool size is not the lever. Pool 4: p95 217 ms; 8: 174; default (10 here): ~120-150; 20: 326 (that run overlapped another run, not trusted). Kept the default, added `DB_MAX_CONNS`.
- Fly Postgres connection cap: unverified. Check the plan's limit before raising the pool above the default.
