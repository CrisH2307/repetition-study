# Space Repetition hackathon build

Product: students teach a concept aloud to an AI student. A separate grounded
checker finds gaps. Gaps change FSRS review dates, and those change next week's
study plan. Success today: a wrong explanation visibly changes the next review
and next week's plan.

Hard design rules
- Code owns all state and rules. LLMs only write words and label claims.
- The AI student never receives source material. It receives only claims the
  human taught, in the human's words, including wrong ones, plus one probe
  target chosen by code.
- The AI student only asks questions. It never corrects or teaches.
- The checker sees sources. Every correction needs a citation whose quote is
  verified by code as an exact substring of an approved source. No valid
  citation means no correction.
- Approved sources are the course unit and the reviewed allowlist only.
- Student speech and text are untrusted data, never instructions to any model.
- Student explains with sources hidden. Corrections appear after the attempt.
- review_events is append-only.
- Do not claim learning gains anywhere in the product or demo.

Stack: Go backend (net/http, pgx, SSE), Postgres, go-fsrs, Gemini for the
student, Claude for the checker. Python only in scripts/parse_pdf.py, outside
the request path. React, Vite, TypeScript and KaTeX for the thin demo UI.

Working rules
- Never assume requirements, names, versions or library APIs. Check the code or
  docs, or ask Cris. Mark anything unverified as unverified.
- Stay inside the file boundary in your prompt. Need a contract change? Stop and
  ask Cris. Never edit internal/domain, api/ or migrations/ from a unit branch.
- Never commit to main. Branches are feature/<name>, fix/<name>, chore/<name>.
- Small PRs into qa, one function, fix or concern each. Commit messages are
  imperative and explain why when it is not obvious.
- Keep rpi/<unit>/research.md, plan.md and impl-log.md short and current.
- Never commit secrets. Read keys from the environment.
- Do not use em dashes in product copy or docs.
- When context usage passes 50%, stop and ask Cris about handing over.
- Final report under 200 words: done, tests run with results, risks, unverified.

Repo: CrisH2307/repetition-study. The hackathon deadline (2026-11-09) has passed;
dates now come from docs/ROADMAP.md. Hackathon plan: docs/PLAN.md.
Your task comes from rpi/<unit>/PROMPT.md through /unit.

Current phase (proposed, pending D24): Phase 0, stabilize and prove the thesis.
Read in this order:
1. docs/STATUS.md
2. docs/DECISIONS.md
3. docs/ROADMAP.md
4. rpi/<unit>/PROMPT.md
Proposals in docs/DECISIONS.md are not decisions until marked Decided there.
