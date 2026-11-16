# P1-WEB plan
1. api.ts: v3 types, one fetch helper that reads `{"error"}` and flags 401 as signed out.
2. App.tsx: sign-in from `/invite/<token>`, signed-out screen, top bar (Today, Courses, Week, Sign out), hash routes. Top bar hidden while teaching.
3. Today.tsx: course folders, minutes, due dates (red within 2 days), Study Time box with bar and reason.
4. Teach.tsx: Teach, Self-grade per claim (stored), Report with reason line, guess next to check, dispute per claim.
5. Course.tsx: courses list and add, notes with status, upload (PDF or pasted text) with polling, page check side by side with image, claim checklist with add, study cards, Mistakes table.
6. Week.tsx: date range, hours with reason, schedule by day with today marked, notes.
7. Mock: in-memory state for all of the above. Tests: vitest units; Playwright `e2e/course.mock.ts` with `playwright.mock.config.ts` (mock only).
Risks: real-API e2e red until P1-API; invite URL shape unverified.
