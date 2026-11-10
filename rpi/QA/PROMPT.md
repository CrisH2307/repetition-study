You own QA on branch chore/qa-run, using the qa branch. Files: qa/ only. Report
bugs, do not fix them. List each with steps to reproduce and severity.

Run
1. go test -race ./... , web tests, and the full-loop test.
2. A load test with the fake LLM: 50 concurrent users.
   Targets: non-LLM reads p95 under 50 ms; schedule and plan recompute under
   20 ms per user. Then 50 concurrent streaming sessions on the fake LLM.
3. With real models, record first-token time (target under 1.5 s) and checker
   result time (target under 6 s) over 10 turns. Say these include vendor latency.
4. The demo acceptance checklist from the plan, item by item.
5. Security checks: no secrets in git history, source text absent from every
   student prompt, and SQL parameterized.
6. A review_events update attempt must fail.

Write qa/report.md with measured numbers and pass or fail for each. Never mark a
target as passed without a measurement. Report in under 200 words.
