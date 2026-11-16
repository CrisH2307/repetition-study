# Plan

This replaces the hackathon plan of 2026-11-09. Why: `docs/improved/strategy-review-fable.md` (called "the review") found the checker was never run on a real model, and Cris and Dino approved its recommendations. Decisions are in `docs/DECISIONS.md`. Rules for agents are in `CLAUDE.md`. How to run agents and which model to use is in `docs/CLAUDE-CODE-SETUP.md`. Testing for Dino is in `docs/TESTING.md`.

## What we are building

A student explains a concept in a derivation-heavy course, typing LaTeX. An AI student only asks questions. A separate checker compares the explanation with the student's own course notes and shows the exact quoted passage for every correction. A gap makes that claim come back sooner, and the week's study budget follows. We do not claim learning gains.

## Where we are

Built and merged into `qa`: contract v2, citation verification (`internal/cite`), Postgres store, scheduler fixes, planner time zones, a vendor-neutral LLM layer with metering and a spend cap, the outage state (`unchecked`, no schedule change), the API handlers, a typed teach page in `web/`, never-taught concepts counting as due, a `fake-down` checker switch, `cmd/evalcheck` running all 8 gate suites, and frozen human labels (Dino and a classmate, kappa 0.964, 2 settled).

Left: real model runs (the live check of the LLM layer, then P0-GATE, #23; steps in `docs/GATE.md`), Dino's page testing (#26, `docs/TESTING.md`), and the UI design (#51). Gate run 1 is recorded in `eval/results/gate-2026-11-15-run1.md`. Gate run 2 (2026-11-16, test split, after D23, D24, a JSON retry and an AI student prompt fix): 6 of 8 rows pass. After the person review in #67 (13 of 13 quotes relevant, l05 not a leak), all 8 rows meet their bar: gate G0 is met on GPT 6 Luna. Report: `eval/results/gate-2026-11-16-run2.md`.

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

## Phase 1: MVP for Dino and classmates (started 2026-11-16)

Goal: five or more students upload their own notes for a quantitative course, check the transcription, list the key claims, teach back, and follow a Today queue on a deployed site, without Cris touching the database. Gate G0 is met on GPT 6 Luna (run 2), so checked reports may change schedules. Decisions D26 to D35. Main agent plan: `rpi/P1-MAIN/plan.md`.

Not in Phase 1: billing, voice, digest, timed calendar blocks, claim-level schedules (D4 stays approved, it comes after the pilot starts), model-made transcription or claim proposals (paid, needs Cris).

Labels: one type label, `phase-1` and `wave-N`. Each unit has `rpi/<unit>/PROMPT.md` and a `.claude/boundaries.json` entry.

| Wave | Unit | Branch | Work | Needs |
| --- | --- | --- | --- | --- |
| 1 | P1-CONTRACT | `feature/contract-v3` | Users and sign-in sessions, uploads and pages, claim checklist, self-grades, disputes, Today and week plan, course view; OpenAPI for all of it | nothing |
| 1 | P1-E2E | `feature/e2e-loop` | UI step 7: Playwright full typed loop against the fake model, in CI | nothing |
| 1 | P1-IMAGE | `chore/docker-image` | One Docker image: API plus built web app plus poppler; compose for a production-like run | nothing |
| 2 | P1-STORE | `feature/pg-store-v3` | Postgres for the new tables, every query scoped to one user | P1-CONTRACT |
| 2 | P1-COURSE | `feature/course-notes` | PDF to pages (pdftotext, pdftoppm), page check, Ready pages become source blocks, claim anchors verified by `internal/cite` | P1-CONTRACT |
| 2 | P1-AUTH | `feature/invite-auth` | Invite links, session cookies, middleware, `cmd/invite` | P1-CONTRACT |
| 2 | P1-TODAY | `feature/today-queue` | Today queue and week plan with minutes and a reason for every number | P1-CONTRACT |
| 3 | P1-API | `feature/api-v3` | Handlers and wiring for all of the above, isolation test, dispute export | wave 2 |
| 3 | P1-WEB | `feature/web-app` | Four screens (Today, Teach, Course, Week) plus sign-in, upload and claim checklist; UI steps 5 and 6 | P1-CONTRACT |
| 4 | P1-DEPLOY | `chore/deploy` | Deploy to the host Cris picks, backups, restore test | Cris's answers, wave 3 |
| 4 | P1-QA | test issue | Dino tests the deployed site | P1-DEPLOY |

Gate G1 (review section 5) is measured during the pilot, after Phase 1 ships.

## Phase 2: pilot tooling (planned 2026-11-16, starts after Phase 1 ships)

Goal: run the pilot with Dino and classmates and measure it honestly: usage and dispute numbers, a privacy export and delete, simple import, a smoother first visit, and disputes that grow the eval set. Gate G2 (review section 5) is judged at the end of the pilot. Decisions D36 to D44. Main agent plan: `rpi/P2-MAIN/plan.md`.

Not built (Cris on #107): study arms (D41) and the email digest (D42). Account deletion is real and removes the user's review history too (D43).

| Wave | Unit | Branch | Work | Needs |
| --- | --- | --- | --- | --- |
| 1 | P2-CONTRACT | `feature/contract-v4` | Feedback answers, imports, consent record, account deletion (D43: one guarded path may delete one user's review_events); OpenAPI | Phase 1 in qa |
| 2 | P2-METRICS | `feature/metrics-export` | Sessions per week, week-4 retention, dispute rate, cost per session; CSV export for Cris | P2-CONTRACT |
| 2 | P2-PRIVACY | `feature/privacy-export` | "Download my data" as JSON; "Delete my account" removes everything of that user (D43) | P2-CONTRACT |
| 2 | P2-IMPORT | `feature/csv-import` | Import concepts and claims from CSV, each claim still needs a verified quote | P2-CONTRACT |
| 2 | P2-EVAL2 | `feature/eval-from-disputes` | Disputes Dino upholds become eval cases on the dev split; script to re-run the gate (Cris runs it, paid) | P2-CONTRACT |
| 3 | P2-API | `feature/api-v4` | Handlers for the v4 endpoints, `cmd/export -deletions`, isolation test | wave 2 |
| 3 | P2-WEB2 | `feature/web-pilot` | First-visit guide, empty states, feedback form, download and delete buttons | wave 2 |

## Cut order if time runs short

Voice input, digest, timed blocks, extra screens. Never cut the citation fix, the outage state, or the gate.

## Open

D16 (name) and D18 (hours each of you can give). For Phase 1: host, sign-in method, domain and the pilot model budget (one question to Cris, see `rpi/P1-MAIN/plan.md`).
