Unit P0-SLICE, wave 3 (can start in wave 1 against `api/openapi.yaml` and the fakes). Branch `feature/dogfood-page`. Model: Sonnet 5.5, effort medium.
Files: `web/`, `rpi/P0-SLICE/`.
Read first: `docs/improved/strategy-review-fable.md` section 4.8 (Teach and Report screens), `api/openapi.yaml`, `docs/DECISIONS.md` (D3, D19).

Build one plain page with React, Vite, TypeScript and KaTeX. Typed input first. No microphone in this unit.

Steps
1. Pick a concept from a list.
2. A text editor with live KaTeX preview. The AI student's single question shows above it. No source text is ever shown while teaching.
3. Stream the question over SSE (`token`, `done`, `error` events). Show a clear error state, never a blank screen.
4. Finish, then a self-grade step (D19): the student ticks which claims they think they covered.
5. Report: each claim with the student's words, the label, the correction, the quoted passage and page, and the visible before and after review date. If the status is `unchecked`, show a banner saying nothing changed.
6. Develop against a mock of the API so it works without the Go server. A Vite proxy to the real server is a config switch.

Rules: formulas always rendered, labels in words and colour, no streaks or lives, accessible contrast. Dino is not technical, so every message in the UI is plain language.
Done when: `npm run build` and any component tests pass, and Dino can complete a session against the mock. Real use against the server waits for P0-INT.
Do not open a PR. Push the branch and stop. Report under 200 words.
