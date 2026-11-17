# Research
- Non-AI reads: GET /v1/today, /v1/week?week=YYYY-MM-DD (400 without the date), /v1/courses, /v1/courses/{id}/view.
- Sign-in: POST /v1/auth/redeem with an invite token; cookie `session`. Invites are reusable (D35), so the tool needs a token list (one per user).
- Fake model: `STUDENT_MODEL=fake CHECKER_MODEL=fake`. Scratch DB `loadtest` in the dev Postgres container, dropped after.
- Go 1.26: loop variables are per iteration.
