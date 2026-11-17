Unit P1-E2E, wave 1. Branch `feature/e2e-loop`. Model: Sonnet 5.5, effort medium. Labels: `Test`, `phase-1`, `wave-1`.
Files: `web/package.json`, `web/package-lock.json`, `web/playwright.config.ts`, `web/e2e/`, `.github/workflows/`, `rpi/P1-E2E/`.
Read first: `docs/design/ui-design.md` (step 7), `docs/TESTING.md`, `web/src/App.tsx`, `cmd/api/main.go`, `.github/workflows/ci.yml`.

UI step 7: one Playwright test of the full typed loop against the real Go API with the fake model and Postgres: pick a concept, type an explanation with a `$` formula, get the AI student's question, finish, self-grade, see the report with before and after dates. A second test with the `fake-down` checker shows the "Not checked yet" banner and no date change.
Add a CI job that builds the web app, starts Postgres, migrates, seeds the fixture pack (never `content/private/`), starts the API on the fake model and runs Playwright (Chromium only).
Done when: `npx playwright test` passes locally and the CI job is green. Push, PR into qa with the labels above, "Closes #<issue>". Never merge. Report under 200 words.
