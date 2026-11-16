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
| D23 | Checker asks the model 3 times per check. Per claim, the label at least 2 of 3 runs agree on wins, with the citation, correction and probe hint from the first run that gave it. No majority, or any failed run, means the report is unchecked and no schedule changes. Why: the G0 dev run with GPT 6 Luna gave 87.5% same-answer-twice stability (bar 95%) even at temperature 0 and a fixed seed, and the 4 label changes in the cheating-attempts suite looked like the same noise. Cost: about 3 times the checker spend, about US$0.008 per session against the US$0.08 bar. Checks take up to 3 times as long (calls run one after another), so the teach check timeout default goes from 20s to 60s. | Approved (Cris, 2026-11-15) |
| D24 | Cheating-attempts bar counts only label changes that help the cheater (a label that ranks better with the attack than without; rank: supported, partial, absent, contradicted; a missing label counts as absent). Harsher changes are still reported but not counted. Bar stays 0. Why: on the dev split every injection change for GPT 6 Luna and Gemini 3.8 Flash made a label harsher, matching checker noise, and a harsher label cannot help an attacker. Decided after seeing dev results and before any test-split run with it. | Approved (Cris, 2026-11-16) |
| D25 | UI build steps 5 (course and study cards, Mistakes table) and 6 (weekly plan) need API contract changes, so they move to Phase 1. | Approved (Cris, 2026-11-16) |
| D26 | Phase 1 scope and waves as in PLAN.md. Units start in their own worktrees from `qa`; only P1-CONTRACT edits `internal/domain`, `api/` and `migrations/`. | Default (main agent, 2026-11-16) |
| D27 | Transcription is free and code-only: `pdftotext` and `pdftoppm` (poppler) split a PDF into pages; the student fixes each page's text next to the page image and marks it Ready. Pasted text works the same way. Only Ready pages become approved source blocks, so only they can be quoted. The PDF is kept in Postgres (limit 20 MB), no file storage service. A model that reads pages is not built (paid, needs Cris). | Default (main agent, 2026-11-16) |
| D28 | Claim checklist: the student writes each key claim of a concept (definition, statement, condition or method, per D20) and anchors it to an exact quote from a Ready page. `internal/cite` verifies the quote; no valid quote, no claim. No model proposes claims in Phase 1. | Default (main agent, 2026-11-16) |
| D29 | Accounts: invite links until Cris picks a sign-in method. Cris makes a link with `cmd/invite`; opening it creates the user and sets an HttpOnly session cookie (random token, stored hashed, 30 days). No passwords, no email service. Every read and write is scoped to the signed-in user, proven by an isolation test. Existing Phase 0 data stays with the default user. | Default (main agent, 2026-11-16) |
| D30 | Self-grades and disputes are stored and never change a schedule. "I think this check is wrong" saves a dispute; the date stays as shown; Dino reads a CSV export (`cmd/export -disputes`). | Default (main agent, 2026-11-16) |
| D31 | Today queue and week plan: deterministic code from due dates, weekly hours and fixed minute estimates (teach 15, recall 5), with a short reason in words for every number. No timed blocks (D13). | Default (main agent, 2026-11-16) |
| D32 | Deployment shape: one Docker image (Go API serves the built web app and runs poppler) plus one Postgres, in a Canadian region (D17), daily backups and a tested restore. The host is Cris's choice; until then the image runs anywhere Docker runs. | Default (main agent, 2026-11-16) |
| D33 | Model use in Phase 1: same GPT 6 Luna roles and the 3-vote check (D22, D23). A monthly app-wide spend cap from the environment; when it is reached checks report `unchecked` and no schedule changes. The cap value is Cris's. | Default (main agent, 2026-11-16) |
| D34 | Product name stays a placeholder ("Space Repetition") on the pilot site, which is invite-only and not public, so D16 is not triggered. | Default (main agent, 2026-11-16) |
| D35 | An invite link keeps working after first use and always signs in the same user (contract v3), so a student can add a second device without Cris. The link works like a password: send it privately. | Default (main agent, 2026-11-16) |
| D36 | Phase 2 scope and waves as in PLAN.md. Pilot metrics come from existing rows (sessions, review events, disputes, model meter); no tracking scripts, no third-party analytics. | Default (main agent, 2026-11-16) |
| D37 | Privacy: "Download my data" gives one JSON file of everything the user wrote and every schedule change. Deleting an account follows D43. | Default (main agent, 2026-11-16) |
| D38 | Import: CSV only (concept, claim, kind, quote). Every imported claim goes through the same quote check as a typed one (D28). Anki import is not built (new dependency). | Default (main agent, 2026-11-16) |
| D39 | Disputes that Dino upholds become eval cases on the development split only. The frozen test split never changes. Re-running the gate is paid and Cris runs it. | Default (main agent, 2026-11-16) |
| D40 | Feedback form: one optional question per week, including "How would you feel if you could no longer use this?" (G2). Answers are stored with the user and never shown to others. | Default (main agent, 2026-11-16) |
| D41 | No study arms and no ethics review for now: the pilot is plain student use (Cris on #107). PL-STUDY is not built. No research claims and no learning-gain claims. | Approved (Cris, 2026-11-16) |
| D42 | No daily email digest in Phase 2 (Cris on #107). No email service. | Approved (Cris, 2026-11-16) |
| D43 | Deleting an account removes everything of that user, including their review history (Cris on #107, option a). This is the one exception to the append-only review log: only a single delete-account code path may remove review_events rows, and only all rows of one user at once; the trigger still rejects every other update or delete. | Approved (Cris, 2026-11-16) |
| D44 | Consent and privacy texts are written in plain words and shown to pilot users without a separate legal review (Cris on #107: "people can have access"). Read as: no named reviewer; Cris corrects this row if he meant something else. | Approved (Cris, 2026-11-16), wording unverified |
| D45 | Hosting: Vercel serves the web app (Cris, 2026-11-16). The Go API (SSE, background PDF reading with poppler) runs as the Docker image on a container host behind Vercel rewrites of `/v1/*`, so the cookie stays same-origin. The container host is still open. | Approved (Cris), API host open |
| D46 | Sign-in stays invite links (D29, D35) for the launch; another method later. Domain later; the Vercel address until then (Cris, 2026-11-16). | Approved (Cris, 2026-11-16) |
| D47 | Launch first: Phase 1 goes to market after a UI polish pass with the ui-ux-pro-max skill. Polish only: no new features, copy or imagery. Phase 2 waits until the launch build is out. Cris tests in place of Dino for now. | Approved (Cris, 2026-11-16) |

## Record of changes to earlier decisions

- Voice first (earlier "confirmed decision") is replaced by D3.
- Concept-level scheduling (earlier) is replaced by D4.
- The outside-source allowlist (Cris's edit to the old plan) is replaced by D14.
- Eval labels by Cris and Dino (earlier) are replaced by D10.
