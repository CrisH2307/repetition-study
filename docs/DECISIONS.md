# Decisions

The one decision log for this repo. Regulations 1, 5 and 10 apply: only Cris or
Dino decide. An item moves from Pending to Decided only when Cris or Dino writes
the answer here with a date. Proposals by Claude (including the Fable 5.2
strategy of 2026-11-09) are not decisions.

How to decide: copy the row to "Decided", fill in Answer, Who and Date, delete it
from "Pending". Mirror the change to the Claude Project docs the same day.

## Decided

Source keys: [DOC] plan doc "Space Repetition Hackathon Plan" tab 1; [C-PLAN]
planning chat 2026-11-09; [C-REG] regulations chat 2026-11-09; [GIT] repo history.

| Topic | Decision | Who | When | Source |
| --- | --- | --- | --- | --- |
| Regulations | Project regulations 1 to 10 adopted (Cris dictated 8 points, Claude wrote 10 rules). Three additions by Claude are unconfirmed, see D8 | Cris | 2026-11-09 morning | [C-REG] |
| First users | Cris, Dino and classmates | Cris | 2026-11-09 | [DOC] Confirmed decisions |
| Repos | Rebuild features, borrow ideas; do not fork or merge the three reference repos | Cris | 2026-11-09 | [DOC] |
| Review unit | Concept, holding cards, problems and teach-back | Cris | 2026-11-09 | [DOC] (D4 proposes a change) |
| Knowledge sources | Course material plus a curated allowlist (allowlist is empty in practice, see D14) | Cris | 2026-11-09 | [DOC] |
| Plan inputs | Hours, level and deadlines first, then measured data | Cris | 2026-11-09 | [DOC] |
| Scheduling | FSRS with manual override | Cris | 2026-11-09 | [DOC] |
| Teaching input | Voice first, typed LaTeX when needed | Cris | 2026-11-09 | [DOC] (D3 proposes a change) |
| Notifications | One daily digest by subject | Cris | 2026-11-09 | [DOC] |
| Backend | Go; Python only for PDF parsing, outside the request path | Cris | 2026-11-09 | [DOC] |
| Demo scope | Closed loop on one course, material pre-loaded | Cris | 2026-11-09 | [DOC] |
| Rules on hackathon day | Written UI exception for the demo only; Git flow unchanged | Cris | 2026-11-09 | [DOC] (exception not logged under regulation 10) |
| AI access | Google Gemini API, Anthropic API, browser speech | Cris | 2026-11-09 | [DOC] |
| Plan approval | "Approve this plan, the proposed gate thresholds and the performance targets" checked | Cris | 2026-11-09 | [DOC] Open Questions |
| Hackathon gate bars | Wrong flagged at least 4 of 5; correct falsely contradicted at most 1 of 10; 100% corrections cited; 0 untaught facts in 10 probes | Cris (via plan approval) | 2026-11-09 | [DOC], rpi/GATE/PROMPT.md |
| Performance targets | Non-LLM p95 under 50 ms at 50 users; recompute under 20 ms; student first token under 1.5 s; checker under 6 s | Cris (via plan approval) | 2026-11-09 | [DOC] |
| Grade mapping | Worst required claim: contradicted or absent gives Again, partial gives Hard, all supported gives Good, never Easy; logged as teach_back | Cris (via plan approval) | 2026-11-09 | [DOC], rpi/U4 |
| AI student role | AI student only asks questions; cited corrections come after the attempt | Cris | 2026-11-09 | [DOC] Open Questions |
| U0 | U0 accepted; "small first, then expand large" | Cris | 2026-11-09 10:43 | [C-PLAN] t8 |
| Demo unit | "Use MATH from Dino, as a test data" (rights not stated, see D7) | Cris | 2026-11-09 | [DOC] Open Questions |
| Roles | "Cris and Dino, Dino done the plan, Cris will handle the code" | Cris | 2026-11-09 | [DOC] Open Questions |
| Repo | CrisH2307/repetition-study, private | Cris | 2026-11-09 | [DOC], GitHub API |
| Age | Users under 18 out of scope | Cris (via plan approval) | 2026-11-09 | [DOC] scope section, not in its Confirmed decisions table; Dino asked it again in [C-DINO], unanswered |

### Implemented but never confirmed by a human

These are in code or prompts today. Treat them as open, not decided.

- Outside-source pack: Cris named LeetCode, NeetCode, AlgoMonster, AlgoExpert;
  Claude deferred them for licence reasons; `content/allowlist.json` is `[]`. See D14.
- Planner constants and the "weekly_hours is a ceiling" reading
  (`internal/planner/allocate.go:152`, rpi/U5/research.md). See D13.
- `EnableShortTerm=false` in FSRS (rpi/U4/research.md). See D4.
- Probe-hint leak guard of 5 shared words (`internal/checker/verify.go:13`). See D6.
- Dino's five questions of 12:40 (language, PDF worker, thin client, 18+, goal):
  no answers recorded. Covered by D1 and D8.
- Jev work split (regulation 3): never run; no manual split logged. See D8.

## Pending Cris and Dino

Every row is a proposal. "Rec." is Claude's recommendation (Fable 5.2 unless
marked new). Claude is a Claude model, so model choices go to the bake-off (D11).

| ID | Question | Options | Rec. and why | Blocks |
| --- | --- | --- | --- | --- |
| D1 | What is this through April 2027? | Validated tool with a pilot; startup now; research study; portfolio piece | Validated tool, business decision at G2. Thesis unproven, market leader free, both are full-time students | Phase 1 scope, D12, D15 |
| D2 | Target segment | Quantitative graduate and upper-year courses in Ontario; all university students; exam prep | Quantitative courses, Ontario first. Generic tools fail visibly there and a real user exists | docs/STRATEGY.md, Phase 1 recruiting |
| D3 | Teaching input | Typed Markdown plus LaTeX first, voice optional; voice first (current decision) | Typed first. Material is formulas; speech was rated High risk in docs/PLAN.md | P0-SLICE, M-WEB |
| D4 | Memory unit | Claim-level FSRS state with concept rollup; concept-level (current) | Claim-level. Matches how FSRS models items; "this gap comes back sooner" becomes literally true | M-CONTRACT3, M-CLAIMSRS (Phase 1) |
| D5 | Checker timing | One final check at Finish plus cheap per-turn matcher; strong check every turn (current) | Final check. About a third of the estimated cost, removes the Finish race | P0-OUTAGE (Finish path), M-CHECK2 |
| D6 | Probe hints | Code-owned probe taxonomy plus approved cue labels; free-text hints from the checker (current); no topic cues | Taxonomy plus cues. Closes the leak channel F2 | M-PROBE (Phase 1); P0-GATE measures the current design |
| D7 | Course material in the repo | Remove from tracked files and history, keep in gitignored `content/private/`, ask the instructor for permission; leave as is | Remove and ask. York states instructors own teaching materials; the plan already said `content/private/` | P0-HYG (content move), P0-EVALDATA (where real-unit data lives) |
| D8 | Regulations | (a) lift "UI deferred" for the MVP or keep; (b) Jev rule keep, optional, or drop; (c) confirm the three additions (branch flow, artifact list, stack list); (d) keep one regulations file | Lift the UI deferral with a logged change; Jev optional with a logged manual split; confirm additions; keep `claude/regulations.md` | P0-SLICE, M-WEB |
| D9 | Gate G0 bars | Adopt the ROADMAP G0 table; keep only the four hackathon bars | Adopt the table, keep the old bars as a smoke check. 4 of 5 is consistent with a true rate of about 28% to 99% | P0-EVALDATA (set sizes), P0-GATE |
| D10 | Labellers | Dino plus a domain classmate; Cris and Dino (current plan) | Dino plus a classmate. Labels need domain knowledge | P0-EVALDATA |
| D11 | Models and bake-off budget | Bake-off across Gemini and Claude tiers for checker, matcher and student, US$50 cap; fixed Gemini student and Claude checker | Bake-off decides, with the cap | P0-LLM (real calls), P0-GATE |
| D12 | Pilot form | Product beta with consent; formal study with ethics review | Beta; ask a supervisor in November whether publishing needs ethics review | Phase 2 |
| D13 | Planner scope in v1 | Today queue plus weekly budgets; timed 30-minute calendar blocks | Today queue. Timed blocks are absent from Dino's real workflow | M-TODAY (Phase 1); P0-TZ is independent |
| D14 | Outside-source allowlist | Drop from v1; pursue LeetCode-style packs | Drop. Licence risk and off-wedge | none in Phase 0 |
| D15 | Pricing test | Free through pilot then test a term pass; freemium; charge now | Free through pilot, test the term pass at G2 | Phase 2 survey |
| D16 | Name | Rename before any public page; keep "Space Repetition" | Rename. Reads as a typo of "spaced repetition" and is hard to search | L-SITE (Phase 3) |
| D17 | Hosting and data location | Canadian region database, disclosed US model processing; other | Canadian region, disclosed | M-DEPLOY (Phase 1) |
| D18 | Capacity | Hours per week each of Cris and Dino commit through January | No recommendation. Roadmap dates are guesses until this is stated | All dates in docs/ROADMAP.md |
| D19 | Self-grade step | Always on; optional; off | Always on: fallback product, control arm, calibration data | M-FEEDBACK (Phase 1) |
| D20 | U1 concept list | Approve the 8 draft concepts; rewrite claims as definitions, statements, conditions and methods with Dino | Rewrite with Dino. Numeric worked answers make poor teach-back targets | P0-EVALDATA |
| D21 | Hackathon units U6, U7, GATE, INT, QA, RELEASE and the `demo-v1` tag (new) | Mark superseded by the roadmap; finish them as written | Superseded. U6 moves to PL-DIGEST, U7 to P0-SLICE and M-WEB, GATE to P0-GATE, INT to P0-INT, QA to M-QA; no `demo-v1` tag. Old GATE and INT boundaries overlap the P0 ones (`eval/results/`, `cmd/`, `internal/app/`) so they must not run in parallel | Which rpi folders stay active |
| D22 | Citation strength (new, from F8) | Minimum quote of 20 characters after normalization, or a whole display-formula block, plus code-derived locator; exact substring only (current) | Minimum length. A one-word quote proves nothing | P0-CITE (min length step) |
| D23 | Checker outage behaviour (new, from F6) | Report status `unchecked`, no review_events row, retry later; mark all required claims absent (current) | `unchecked`. A vendor incident must never change a schedule | P0-CONTRACT (status field), P0-OUTAGE |
| D24 | Phase 0 plan approval (new) | Approve docs/ROADMAP.md Phase 0 units, owners and dates; amend | Approve or amend before any Phase 0 unit implements (regulation 5) | Every P0 unit |

## Change log

- 2026-11-09: File created from the planning dossier and the Fable 5.2 strategy.
  All Pending rows are proposals. (Claude, on chore/roadmap-v2; not yet reviewed.)
