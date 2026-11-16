Unit P1-WEB, wave 3. Branch `feature/web-app`. Model: Opus 5.5, effort medium. Labels: `User Story`, `phase-1`, `wave-3`.
Files: `web/`, `rpi/P1-WEB/`.
Read first: `docs/design/ui-design.md` (all screens, look, decisions), `api/openapi.yaml` v3, `web/src/`.
Four screens with a small top bar: Today (screen 1), Teach with Self-grade and Report (screens 2 to 5, already built, now with the reason line, "I think this check is wrong" and stored self-grade), Course (screen 6, UI step 5: notes with status, upload, page check side by side with the page image, claim checklist, study cards, Mistakes table), Week (screen 7, UI step 6). Plus a sign-in screen for an invite link and a plain "signed out" state. Develop against the mock; extend the Playwright test from P1-E2E with upload, page Ready, add a claim, teach, report.
Plain words everywhere, formulas rendered with KaTeX, touch targets at least 44px, works at phone width. No streaks, no learning-gain claims.
Done when: `npm run build`, unit tests and Playwright green. PR into qa with the labels above. Never merge. Report under 200 words.
