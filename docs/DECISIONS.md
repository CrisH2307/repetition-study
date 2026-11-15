# Decisions

One log. Newest change at the top of each row. On 2026-11-09 Cris and Dino approved the Fable strategy review (`docs/improved/strategy-review-fable.md`, section 6) as a whole. Every row below is approved as the review recommended unless marked open. Cris corrects any row by editing it here.

| # | Decision | Status |
| --- | --- | --- |
| D1 | Goal through April 2027: a validated tool with a pilot. Business decision at gate G2. | Approved |
| D2 | Target: quantitative graduate and upper-year courses, Ontario first. | Approved |
| D3 | Input: typed LaTeX with live preview first. Voice is optional and fills the same box. | Approved |
| D4 | Memory unit: one FSRS state per claim, rolled up to the concept for display and planning. | Approved |
| D5 | Checker: one final check at Finish, plus a cheap per-turn matcher that never sees source prose. | Approved |
| D6 | Probes: code-owned question types plus human-approved cue labels. No free-text hints from any model that saw the source. | Approved |
| D7 | Course material: purge from the repo and its history, keep in gitignored `content/private/`, ask the instructor for written permission. Cris runs the history purge by hand. | Approved |
| D8 | Regulations: "UI deferred" is lifted for the MVP. Jev is optional, log a manual split when not run. One regulations file. Flow: issue, branch from `qa`, agent opens a PR into `qa`, Cris merges, Dino tests on `qa`, Cris merges `qa` into `main`. Agents never merge. | Approved |
| D9 | Gate bars: adopt the table in PLAN.md. The old four bars stay as a smoke check. | Approved |
| D10 | Eval labels: Dino plus one classmate, not Cris. Cris builds tooling. | Approved |
| D11 | Models: a bake-off decides student, matcher and checker across both vendors, cap US$50. No role is fixed in advance. | Approved |
| D12 | Pilot form: product beta with consent. Ask a supervisor in November whether ethics review is needed before publishing. | Approved |
| D13 | Planner in v1: a Today queue and weekly budgets. No timed calendar blocks. | Approved |
| D14 | Outside-source allowlist (LeetCode and similar): dropped from v1. | Approved |
| D15 | Pricing: free through the pilot. Test a term pass at gate G2. | Approved |
| D16 | Name: rename before any public page. New name not chosen. | Open |
| D17 | Hosting: Canadian region database, US model processing disclosed. | Approved |
| D18 | Capacity: hours per week each of Cris and Dino can give through January. | Open |
| D19 | Self-grade step before the report: always on. | Approved |
| D20 | Course claims: rewrite as definitions, statements, conditions and methods, with Dino. Worked numeric answers are not teach-back targets. | Approved |
| D21 | Real model calls go through OpenRouter (one key, one prepaid balance as the hard limit). Same roles: Claude checker, Gemini student. Ids with the `openrouter/` prefix route there; bare `claude-*` and `gemini-*` ids still mean the direct APIs. | Approved |
| D22 | Model: GPT 6 Luna via OpenRouter for both the AI student and the checker (cost). Replaces "Gemini for the student, Claude for the checker". The gate measures it; if G0 fails, one retry with a different checker model as PLAN says. | Approved (Cris, 2026-11-14) |

## Record of changes to earlier decisions

- Voice first (earlier "confirmed decision") is replaced by D3.
- Concept-level scheduling (earlier) is replaced by D4.
- The outside-source allowlist (Cris's edit to the old plan) is replaced by D14.
- Eval labels by Cris and Dino (earlier) are replaced by D10.
