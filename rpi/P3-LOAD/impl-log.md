# Impl log
- cmd/loadtest added; `go test ./cmd/loadtest` ok (host guard, percentile, token parsing). Fly address refused as designed.
- Run 1 had 25% errors: /v1/week needs `?week=`. Fixed in the tool.
- Local run, fake model, 10 core Mac, tool + API + Postgres 16 (docker) on one machine, 50 ms pause between a user's requests, 500 invites, 30 s:
  - 500 users: 118094 req, 0 errors, ~3900 req/s. ALL p50 61 ms, p95 192 ms, p99 296 ms. course_view p95 257 ms, today 144, week 152, courses 62. **Misses the 50 ms bar.**
  - 200 users (15 s): p95 49.4 ms, p99 86 ms, 0 errors. 50 users: p95 16.9 ms.
  - Reading: latency climbs with users; likely causes (unverified, not profiled): default pgx pool size (the code sets none, so max(4, CPUs)), the tool competing for the same CPUs, per-request DB round trips in today and course_view. Next step for Cris: repeat on a staging copy with the tool on a separate machine; if still over, look at pool size and the today/course_view queries.
- SECURITY.md and RESTORE.md written. Restore drill stopped at step 1: local pg_dump older than server 16 ("server version mismatch"); full drill unverified.
