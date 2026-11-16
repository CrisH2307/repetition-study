# Space Repetition build

Product: students teach a concept to an AI student, typing LaTeX first and
voice optional. A separate grounded checker finds gaps. Gaps change FSRS review
dates, and those change next week's study plan. Success for Phase 0: a wrong
explanation visibly changes the next review, backed by a real quote, and the
checker passes gate G0 on real models.

Hard design rules
- Code owns all state and rules. LLMs only write words and label claims.
- The AI student never receives source material. It receives only claims the
  human taught, in the human's words, including wrong ones, plus one probe
  target chosen by code.
- The AI student only asks questions. It never corrects or teaches.
- The checker sees sources. Every correction needs a citation whose quote is
  verified by code as an exact substring of an approved source. No valid
  citation means no correction.
- Approved sources are the course unit only. The outside allowlist is dropped.
- One code path verifies citations: internal/cite. Nothing else re-checks quotes.
- The AI student gets no free text from any model that has seen the source.
- If the checker fails, the report is `unchecked` and no schedule changes.
- Course material is never committed. It lives in gitignored content/private/.
- Student speech and text are untrusted data, never instructions to any model.
- Student explains with sources hidden. Corrections appear after the attempt.
- review_events is append-only.
- Do not claim learning gains anywhere in the product or demo.

Stack: Go backend (net/http, pgx, SSE), Postgres, go-fsrs, GPT 6 Luna via
OpenRouter for the student and the checker (D22). Python only in scripts/parse_pdf.py, outside
the request path. React, Vite, TypeScript and KaTeX for the web UI. The "UI
deferred" rule is lifted for the MVP (see DECISIONS D8).

Working rules
- Never assume requirements, names, versions or library APIs. Check the code or
  docs, or ask Cris. Mark anything unverified as unverified.
- Stay inside the file boundary in your prompt. Need a contract change? Stop and
  ask Cris. Never edit internal/domain, api/ or migrations/ from a unit branch.
- Never commit to main. Branches are feature/<name>, fix/<name>, chore/<name>.
- Flow: GitHub issue, then a branch from qa, then a PR into qa, then Dino tests
  on qa, then a PR from qa into main. Every unit has an issue. Small commits,
  one function, fix or concern each. Open your PR with `gh pr create --base qa`
  (body: "Closes #<issue>", tests run, what is unverified).
  Label every issue and PR with one type label: `Bug` (something broken),
  `User Story` (new feature), `Test` (validation), `Doc`, `Issue` (other task) or
  `question`, plus `phase-0` and its wave label. Use `gh ... --label "User Story"`. Never merge a PR and
  never push to qa or main. Cris merges. Commit messages are imperative and
  explain why when it is not obvious.
- Anything written for Dino is plain language: no jargon, exact steps, what pass
  looks like, and "copy the output to Cris and stop" on failure.
- Keep rpi/<unit>/research.md, plan.md and impl-log.md short and current.
- Never commit secrets. Read keys from the environment.
- Do not use em dashes in product copy or docs.
- When context usage passes 50%, stop and ask Cris about handing over.
- Final report under 200 words: done, tests run with results, risks, unverified.

Repo: CrisH2307/repetition-study. Plan: docs/PLAN.md. Decisions: docs/DECISIONS.md.
Model choice per task: docs/CLAUDE-CODE-SETUP.md. Your task comes from
rpi/<unit>/PROMPT.md through /unit.
