# Research
- Stack versions installed: React 19, Vite 8, Vitest 5, KaTeX 0.19, TypeScript 7 (from npm, checked by `npm ls`).
- POST SSE cannot use EventSource, so `web/src/api.ts` reads the fetch stream and parses events itself.
- Contract gaps (not fixed here, outside boundary):
  1. No endpoint lists concepts. The picker uses `GET /v1/reviews/due`, so a concept never reviewed (such as the fixture ones) does not appear against the real server.
  2. No self-grade endpoint, and the report has no claim text. Self-grade is client only: the student ticks their own sent points, and the report shows "ticked N of M".
  3. Per-claim review dates do not exist; the report has one concept-level before and after date.
