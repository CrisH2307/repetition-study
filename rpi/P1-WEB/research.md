# P1-WEB research
- Contract: api/openapi.yaml v0.3.0. Screens use /v1/me, /v1/auth/redeem, /v1/auth/sign-out, /v1/today, /v1/week, /v1/courses, /v1/courses/{id}/view, uploads, pages, blocks, concepts/{id}/claims, self-grades, disputes.
- Invite link URL shape: not in the contract and P1-AUTH is not pushed yet. The web app reads `/invite/<token>` (unverified; P1-AUTH must print links in that shape).
- Report reason line: Report has no reason field. The app shows `schedule.payload.reason` when it is a string, else a line computed from the labels (unverified which one P1-API sends).
- Self-grade needs claim ids, so Self-grade lists the concept's claim checklist (GET /concepts/{id}/claims), shown after the attempt only. With no claims it falls back to the sent points and saves nothing.
- Citation has no page id, so the report shows the page as text; the Mistakes table links to the page (Mistake.page_id).
- Quotes for a new claim are not checked in the browser: internal/cite is the one checker; a 400 shows the server's plain words.
- TodayItem has no card id, so Start opens a teach session for both teach and recall items.
- `.github/workflows/e2e.yml` runs loop.e2e.ts against the Go API. That API has no /v1/me or /v1/today until P1-API lands, so it will be red until then.
- Deps already installed: React 19, Vite 8, KaTeX 0.19, Playwright 1.65, Vitest 5. No new deps.
