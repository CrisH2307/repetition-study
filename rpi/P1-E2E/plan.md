# P1-E2E plan
- web/playwright.config.ts starts two Go APIs (fake, fake-down) and two `vite preview` servers on ports 18080/18081/15173/15174.
- web/e2e/loop.e2e.ts: typed loop with dates; fake-down banner with no date change. Named .e2e so vitest ignores it.
- .github/workflows/e2e.yml: Postgres, migrate, seed fixture pack only, build, Playwright (Chromium).
- Run locally: Postgres up, `make migrate`, `go run ./cmd/seed -pack content/fixture/unit.json`, then `cd web && npx playwright test`. Needs a fresh DB (each topic is taught once).
- Risks: CI not yet seen green; tests need a fresh seeded DB.
