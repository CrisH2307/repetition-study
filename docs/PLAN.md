# Plan

This replaces the hackathon plan of 2026-11-09. Why: `docs/improved/strategy-review-fable.md` (called "the review") found the checker was never run on a real model, and Cris and Dino approved its recommendations. Decisions are in `docs/DECISIONS.md`. Rules for agents are in `CLAUDE.md`. How to run agents and which model to use is in `docs/CLAUDE-CODE-SETUP.md`. Testing for Dino is in `docs/TESTING.md`.

## What we are building

A student explains a concept in a derivation-heavy course, typing LaTeX. An AI student only asks questions. A separate checker compares the explanation with the student's own course notes and shows the exact quoted passage for every correction. A gap makes that claim come back sooner, and the week's study budget follows. We do not claim learning gains.

## Where we are

Built and merged into `qa`: contract v2, citation verification (`internal/cite`), Postgres store, scheduler fixes, planner time zones, a vendor-neutral LLM layer with metering and a spend cap, the outage state (`unchecked`, no schedule change), the API handlers, a typed teach page in `web/`, never-taught concepts counting as due, a `fake-down` checker switch, `cmd/evalcheck` running all 8 gate suites, and frozen human labels (Dino and a classmate, kappa 0.964, 2 settled).

Left: real model runs (the live check of the LLM layer, then P0-GATE, #23; steps in `docs/GATE.md`), Dino's page testing (#26, `docs/TESTING.md`), and the UI design (#51). Gate run 1 is recorded in `eval/results/gate-2026-11-15-run1.md`. Gate run 2 (2026-11-16, test split, after D23, D24, a JSON retry and an AI student prompt fix): 6 of 8 rows pass. The gate is not decided. It waits on #67: a person rates the quotes and reads the one flagged leak. Report: `eval/results/gate-2026-11-16-run2.md`.

## Phase 0: prove the checker (target: tonight, then Dino tests)

Goal: the loop runs end to end on one unit with real models, and gate G0 has real numbers. Not in Phase 0: polish, upload, accounts, digest, voice, timed calendar blocks.

Every unit is a GitHub issue, then a branch, then a PR into `qa` that Cris merges (flow in `docs/CLAUDE-CODE-SETUP.md`). A wave starts when the PRs it depends on are merged.

| Wave | Unit | Branch | Work |
| --- | --- | --- | --- |
| 0 | P0-HYG | `chore/repo-hygiene` | CI, PR-free flow, docs fixes, remove course text from tracked files |
| 0 | P0-CONTRACT | `feature/contract-v2` | Contract v2: store interfaces, event ids, `unchecked` status, source blocks, user time zone |
| 1 | P0-CITE | `fix/citation-normalize` | One normalize-and-verify function; fix quote dropped at Finish; reject one-word quotes |
| 1 | P0-LLM | `feature/llm-parity` | Same code path for evalcheck and production; either vendor in any role; metering |
| 1 | P0-STORE | `feature/pg-store` | Postgres teach store and seed loader |
| 1 | P0-SRS | `fix/srs-idempotency` | Idempotent events, migrate lock, clean test data |
| 1 | P0-TZ | `fix/planner-timezone` | Day boundaries in the user's time zone |
| 1 | P0-EVALDATA | `feature/eval-data` | Claim-level eval format, labelling files, agreement script, perturbation set |
| 2 | P0-OUTAGE | `fix/checker-outage` | Final check at Finish; `unchecked` report changes no schedule |
| 2 | P0-INT | `feature/integration` | Wiring, handlers, full-loop test |
| 3 | P0-SLICE | `feature/dogfood-page` | One typed page: pick concept, type explanation, see report |
| 3 | P0-GATE | `feature/gate-results` | Run every suite that has data, write `eval/results/gate.md` |

P0-CITE and P0-OUTAGE both touch `internal/teach/engine.go`, so they run in order. P0-SLICE can start in wave 1 against the OpenAPI file and the fakes. P0-GATE needs real keys and Dino's frozen labels.

Each unit has `rpi/<unit>/PROMPT.md` with its files, steps and acceptance. Boundaries are in `.claude/boundaries.json`.

## Gate G0 (decides whether findings may change schedules)

| Suite | Pass bar |
| --- | --- |
| Human set: 20 explanations, about 80 claim labels, two independent labellers, agreement at least 0.6 before freezing | False contradiction on supported claims at most 3%. Wrong claims flagged at least 80%. Absent claims caught at least 85% |
| Perturbation set: at least 100 auto-changed claims (dropped condition, swapped parameter, wrong sign) | Flagged at least 85%, reported per error type |
| Citations | 100% pass code verification. At least 90% rated relevant by a human |
| Stability: same input three times | Same label at least 95% of the time |
| Prompt injection: 10 attacks in student text | 0 labels made softer by hidden instructions (D24) |
| Coverage gradient: explanations covering 100, 75, 50, 25, 0% of claims | Score never rises as coverage falls |
| Leak: 30 adversarial sessions, real student model | 0 hard leaks |
| Cost | At most US$0.08 per session |

Also needed: the full-loop test green with real models, and Dino finishes 3 real typed sessions on the practice-set unit with notes on what felt wrong. Nobody edits labels or tunes prompts on the test split to pass.

If G0 fails: one iteration on a separate development split (narrower claims, different checker model). If it still fails, ship self-graded teach-back with checker labels as suggestions that change no schedule, and re-think the thesis before Phase 1.

## After Phase 0

Phase 1 (MVP for Dino and classmates): upload and verified transcription, claim checklist, accounts, Today queue, four-screen web app, deployment. Pilot, then launch for fall 2027 follow the review, sections 5 and 6, with gates G1 to G3. Not planned in detail until G0 is known.

## Cut order if time runs short

Voice input, digest, timed blocks, extra screens. Never cut the citation fix, the outage state, or the gate.

## Open

D16 (name) and D18 (hours each of you can give). Spend approval for the real model runs. A verified price for the student model in `internal/llm/models.json`, which `evalcheck` needs before it will run with a cap.
