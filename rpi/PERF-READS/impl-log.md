# Impl log
Setup: laptop 10 cores, Postgres 16 in docker (scratch DB), fake model, 500 invites, 50 ms pause, 20 s per run. Tool GOMAXPROCS=4, API GOMAXPROCS=2 (macOS has no CPU pinning, so this only limits Go threads; Postgres and the tool still share the machine). Two runs each, before and after.

p95 in ms (run 1 / run 2):
| users | endpoint | before | after |
|---|---|---|---|
| 50 | ALL | 9.8 / 10.5 | 14.3 / 16.2 |
| 200 | ALL | 20.6 / 20.2 | 18.7 / 21.8 |
| 500 | ALL | 121 / 119 | 90 / 93 |
| 500 | course_view | 151 / 159 | 111 / 107 |
| 500 | today | 88 / 88 | 95 / 93 |
| 500 | week | 88 / 80 | 88 / 92 |
500 users p50/p99 ALL: before 50 / 157-198, after 42-45 / 142-147.

Result: course_view and overall improve about 25%; today and week are unchanged. **The 50 ms bar at 500 users is still missed locally.** 50 users reads slightly worse after, within run noise (unverified). Today/week need fewer round trips in StudyState (next step). The laptop is not production.
