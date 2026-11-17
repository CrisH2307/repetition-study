# Research
- Behind the Vercel rewrite, `Fly-Client-IP` is a Vercel edge address, so one bucket covers all students.
- Vercel docs (request-headers, 2025-12-13): `x-vercel-forwarded-for` is the client IP, same as `x-forwarded-for`. Unverified: whether Vercel overwrites a client-sent value on a rewrite to an external origin. So the header is not trusted alone.
