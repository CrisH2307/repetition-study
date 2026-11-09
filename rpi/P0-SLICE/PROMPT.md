You own P0-SLICE on branch feature/dogfood-page, from qa after P0-INT is merged.
Owner: Claude; Dino reviews. Files: web/, rpi/P0-SLICE/.

Read first: docs/STATUS.md, docs/DECISIONS.md, docs/ROADMAP.md.
Decisions: D8 (lifting "UI deferred" must be logged first), D3 (typed first),
D24. Do not start until D8 and D3 are Decided; if D3 is decided as voice first, stop
and ask Cris. State the React/Vite/TypeScript choice, its reason and trade-offs
in plan.md (regulation 7). Write plan.md, then STOP until Cris
approves it (regulation 5).

Build one plain page for dogfooding, not a product UI:
1. Pick a concept from the seeded pack.
2. Type an explanation in Markdown with LaTeX and a live KaTeX preview; sources
   hidden. The AI student's single question shows above the box; replies stream
   over SSE.
3. Finish shows the report: each claim, label, correction, and the verified quote.
   An unchecked report says so plainly.
Use React, Vite, TypeScript and KaTeX with exact pinned versions. Build against
api/openapi.yaml. No learning-gain claims in any copy.

Tests
- Component test for the report view, including unchecked.
- One Playwright run of the typed loop against the fake model.

Done when: Dino completes 3 real sessions on the practice-set unit and writes notes
on what felt wrong into rpi/P0-SLICE/impl-log.md.

Report in under 200 words: done, tests with results, risks, unverified.
