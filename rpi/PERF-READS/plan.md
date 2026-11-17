1. `DB_MAX_CONNS` env for the pool (unset keeps pgx default).
2. course_view: drop the duplicate course lookup, load all cards' claims in one query (9 to 5 round trips).
3. Re-run cmd/loadtest at 50, 200, 500.
Not done: today/week still 5 round trips (StudyState is 4 queries); next step if Cris wants it.
