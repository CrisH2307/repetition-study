# P1-WEB impl log
- api.ts: v3 types, one fetch helper (plain-words errors, 401 means signed out). tsc OK.
- App.tsx: invite sign-in at /invite/<token>, signed-out and bad-link screens, top bar, hash routes, Today with Study Time, Week. Top bar hidden while teaching.
- Teach.tsx: Self-grade per key idea, saved with POST self-grades; Report reason line, guess next to each result, "I think this check is wrong" sends a dispute; unchecked shows "Not checked" and no corrections.
- Course.tsx: courses list and add, notes with status and polling, upload (PDF or pasted text), page check side by side, key ideas with quote from a Ready passage, study cards, add topic, Mistakes table linking to the page.
- mock/plugin.ts: in-memory fake of every v3 path the app uses.
- Tests: vitest 9/9 pass. `npx playwright test -c playwright.mock.config.ts` 6/6 pass (mock only). `npm run build` OK.
- loop.e2e.ts (real Go API): only changed to pick the first "Right". Not run: needs P1-API (/v1/me, /v1/today) and a signed-in user.
