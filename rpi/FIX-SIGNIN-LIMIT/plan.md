# Plan
- Sign-in bucket key: first `X-Vercel-Forwarded-For` entry, else `Fly-Client-IP`.
- Second cap on `Fly-Client-IP`, `RATE_SIGNIN_EDGE_PER_MIN`, default 120 (12x the student cap: a class on one edge fits, a header rotator is stopped after 120 a minute).
- Writes and turns unchanged.
