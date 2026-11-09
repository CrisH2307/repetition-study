# Roadmap

Status: PROPOSAL from the Fable 5.2 strategy (2026-11-09), pending Cris and Dino
(D24 in docs/DECISIONS.md). Every bar, date and scope line below is proposed until
marked Decided there. Dates assume part-time work; capacity is unknown (D18).

Owners: Cris (code owner, contract, merges), Dino (domain labels, design, QA,
user), Claude (Claude Code sessions under Cris, one unit per session).

Every unit: branch from `qa`, entry in `.claude/boundaries.json`,
`rpi/<UNIT>/PROMPT.md` plus research, plan, impl-log; plan approved by Cris
before implementation; small PRs into `qa`; Cris merges. Only a contract unit
touches `internal/domain/`, `api/` or `migrations/`, and Cris owns it.

## Phase 0: stabilize and prove the thesis (proposed 10 Nov to 23 Nov 2026)

**Goal.** The loop runs end to end on one unit with real models, and gate G0 has
real numbers.

**Non-goals.** UI polish, PDF upload, accounts, digest, claim-level scheduling,
probe taxonomy, planner changes beyond the time-zone fix, any public page.

### Units

Start with `scripts/unit.sh <UNIT>` then `/unit <UNIT>` runs in the worktree.

| Unit | Branch | Owner | Work | Needs units | Needs decisions | Acceptance | Proposed dates |
| --- | --- | --- | --- | --- | --- | --- | --- |
| P0-HYG | chore/repo-hygiene | Cris (Claude drafts) | CI, PR template, branch protection, TESTING.md fixes, stale deadline text in skills, move course material per D7 | none | D7, D8 | Direct push to `qa` rejected; CI green on a PR; practice-set text absent from tracked files on `qa` | 10 to 12 Nov |
| P0-CONTRACT | feature/contract-v2 | Cris | Contract v2: pack type, TeachStore, GetReport, event ids, `DueItem.Reason`, report status `unchecked`, user time zone | none | D23 | `go build ./...` and `go test ./...` green; fakes updated; OpenAPI passes a real validator | 10 to 13 Nov |
| P0-CITE | fix/citation-normalize | Claude, Cris reviews | One cite package used by checker and teach (F1); minimum quote length (F8) | none | D22 (length step only) | Multi-line quote survives Finish; one-word quote rejected | 10 to 12 Nov |
| P0-LLM | feature/llm-parity | Claude with Cris's keys | evalcheck uses `internal/llm` (F9); JSON on both vendors; Gemini config (F10); metering; cost per session; leak suite mode; F14 items | none | D11 | `evalcheck --model fake` and one real model per vendor run through `internal/llm`; cost per session printed | 10 to 14 Nov |
| P0-EVALDATA | feature/eval-data | Dino and a classmate label; Claude builds tooling | Claim rewrite, 20 explanations, two label files, kappa script, frozen split, perturbation set, leak probes | none (labels need D20 first) | D7, D9, D10, D20 | Kappa at least 0.6 (proposed); files frozen; hash in docs/DECISIONS.md | 11 to 20 Nov |
| P0-SRS | fix/srs-idempotency | Claude | Advisory lock in migrate; srs tests leave no rows (F12) | none (event id half after P0-CONTRACT) | none | Second migrate run concurrent is safe; test run leaves no rows | 11 to 13 Nov |
| P0-STORE | feature/pg-store | Claude | Postgres TeachStore; approved-source filter (F3) | P0-CONTRACT | none | Restart mid-session, Finish still works; `claim_labels` rows exist | 13 to 17 Nov |
| P0-TZ | fix/planner-timezone | Claude | Day windows in the user's zone (F5) | P0-CONTRACT (zone field) | none | Toronto blocks fall in configured local hours across a DST change | 13 to 16 Nov |
| P0-OUTAGE | fix/checker-outage | Claude | `unchecked` report, no event on outage, event id at Finish (F6, F7) | P0-CITE, P0-CONTRACT | D23, D5 | Checker failing: report `unchecked`, no `review_events` row | 14 to 18 Nov |
| P0-INT | feature/p0-integration | Cris with Claude | Composition root, seed loader, handlers for teach, report, due, review, override; full-loop test | P0-STORE, P0-OUTAGE, P0-SRS, P0-TZ, P0-LLM | none | Full-loop test green with fakes and once with real models | 17 to 21 Nov |
| P0-GATE | feature/p0-gate | Cris runs, Dino reads, both decide | Run every suite with data; write `eval/results/gate.md` | P0-LLM, P0-EVALDATA, P0-CITE (loop item: P0-INT) | D9, D11 | Numbers against bars, every failed example in full | 20 to 22 Nov |
| P0-SLICE | feature/dogfood-page | Claude, Dino reviews | One plain typed page: pick concept, explain, see report | P0-INT | D3, D8 | Dino completes 3 real sessions with written notes | 20 to 23 Nov |

### Dependency order

```
Decisions first: D24, then D7, D9, D10, D11, D20, D22, D23 (D3, D8 before P0-SLICE)

Wave 1 (parallel):  P0-HYG   P0-CONTRACT   P0-CITE   P0-LLM   P0-SRS   P0-EVALDATA (runs through wave 3)
Wave 2:             P0-STORE (after CONTRACT)   P0-TZ (after CONTRACT)   P0-OUTAGE (after CITE and CONTRACT)
Wave 3:             P0-INT (after STORE, OUTAGE, SRS, TZ, LLM)
Wave 4:             P0-GATE (after LLM, EVALDATA, CITE; loop item after INT)   P0-SLICE (after INT)
Decision:           G0 review by Cris and Dino, proposed 23 Nov
```

Serial pairs because files overlap: P0-CITE then P0-OUTAGE (`internal/teach/`);
P0-HYG then P0-EVALDATA if D7 moves `content/`.

### Gate G0 (proposed bars, D9)

| Suite | Data | Proposed bar |
| --- | --- | --- |
| Hackathon smoke bars (Decided) | 20 explanations | Wrong flagged at least 4 of 5; correct falsely contradicted at most 1 of 10; 100% cited; 0 untaught facts in 10 probes |
| Checker, human set | 20 explanations, about 80 claim labels, kappa at least 0.6 | False contradicted on supported claims at most 3%; wrong claims flagged at least 80%; absent claims caught at least 85% |
| Checker, perturbation set | At least 100 perturbed claims, 20% spot-checked | Flagged at least 85%, reported per error type |
| Citation quality | Every correction in the human set | 100% pass code verification; at least 90% rated relevant by Dino |
| Stability | Human set, 3 runs | Label agreement at least 95% |
| Prompt injection | 10 attacks | 0 label changes |
| Coverage gradient | Synthetic 100, 75, 50, 25, 0% coverage per concept | Monotone per concept; Spearman at least 0.9 |
| Leak | 30 adversarial sessions, real student model | 0 hard leaks (0 of 30 still allows about 10% at 95% confidence) |
| Cost | All runs | At most US$0.08 per session |
| Loop | P0-INT full-loop test | Green with real models |
| Use | Dino, P0-SLICE | 3 sessions done with written notes |

Rule kept from `rpi/GATE/PROMPT.md`: no prompt tuning or label edits to pass;
tune only on the development split.

**If G0 fails (proposal):** one iteration on the development split (narrower
claims, different checker model). If it fails again, ship self-graded
teach-back with checker labels as suggestions that never change the schedule,
and re-evaluate the thesis before Phase 1.

## Phase 1: MVP for Dino and classmates (proposed 24 Nov 2026 to 10 Jan 2027)

**Goal.** Five or more students upload their own notes for a quantitative course,
teach back, and trust the schedule, without Cris touching the database.
**Non-goals.** Billing, mobile app, digest, pack sharing, voice, per-user FSRS
fitting, public marketing.

| Unit | Branch | Owner | Acceptance (proposed) | Decisions |
| --- | --- | --- | --- | --- |
| M-CONTRACT3 | feature/contract-v3 | Cris | Courses, enrollments, claim states, disputes, problems, jobs; migration applies on a Phase 0 copy with no data loss | D4 |
| M-INGEST | feature/ingest-math | Claude; Dino supplies gold pages | At least 18 of 19 damaged regions right before human fix; 13 pages ready to verify in under 5 minutes | D7 |
| M-PACK | feature/pack-builder | Claude; Dino judges | At least 80% of proposed required claims accepted unedited on 3 units | D20 |
| M-CHECK2 | feature/checker-final | Claude | Two-stage check, block citations, adjudication; G0 bars still met | D5, D22 |
| M-PROBE | feature/probe-taxonomy | Claude | Leak suite 0 of 30; Dino rates at least 70% of probes useful | D6 |
| M-CLAIMSRS | feature/claim-schedule | Claude | One missing claim moves only that claim; under 20 ms per 1,000 items | D4 |
| M-TODAY | feature/today-queue | Claude | Deterministic; a reason for every change | D13 |
| M-AUTH | feature/auth | Cris | User A cannot read user B through any endpoint | D17 |
| M-FEEDBACK | feature/disputes | Claude | Every dispute in an export Dino can read | D19 |
| M-STORE2, M-API | feature/pg-store-v3, feature/api-v1 | Cris with Claude | OpenAPI and handlers match (contract test) | none |
| M-WEB | feature/web-app | Claude; Dino owns design review | Four screens (Today, Teach, Report, Course); Playwright loop green; Dino signs off | D3, D8 |
| M-DEPLOY | chore/deploy | Cris | Restore test passes; spend cap proven | D17 |
| M-QA | chore/qa-mvp | Dino | No high-severity defect open | none |

**Gate G1 (proposed):** Dino at least 12 sessions over 3 weeks on 2 courses; at
least 4 other students with 4 sessions each within 2 weeks; disputes on at most
10% of corrections, at least 70% of reviewed disputes upheld for the checker;
at most 10 minutes of fixing per 10 pages; zero cross-user exposures; zero
schedule changes from unchecked reports; median cost at most US$0.08 per
session; instructor permission or a recorded decision on the personal-study
position.

## Phase 2: pilot (proposed 11 Jan to 30 Apr 2027)

**Goal.** Learn whether students keep using it and whether there is any
measurable effect, with honest statistics.
**Non-goals.** Paid acquisition, new domains, mobile app.

Units: PL-STUDY (Cris; within-student arms, consent, pre-registered plan, D12),
PL-METRICS (Claude), PL-DIGEST (Claude; replaces U6), PL-PRIVACY (Cris; human
legal review), PL-IMPORT (Claude), PL-WEB2 (Claude, Dino reviews), PL-EVAL2
(Claude, Dino adjudicates).

**Gate G2 (proposed):** at least 20 active in week 4, week-4 retention at least
35%; median at least 2 sessions a week; at least 40% "very disappointed"; G0
bars still met on the grown set; delayed-test estimate published with a 95%
interval (stop and redesign if negative and excluding zero); cost at most
US$1.50 per active user per month; at least 25% say they would pay CA$8 a month
or CA$29 a term, or 10 pre-orders. Fewer than three of the first five met: no
launch; choose a write-up or a pivot.

## Phase 3: market launch (proposed May to Sept 2027)

**Goal.** A paid product for the fall 2027 term in Ontario, with published evidence.
**Non-goals.** K-12, users under 18, native apps, enterprise features.

Units: L-BILLING (Cris), L-SHARE (Claude, Cris reviews rights logic),
L-INSTRUCT (Claude), L-SITE (Dino copy and design, Claude builds; D16), L-WEB3
(Claude, Dino reviews), L-FSRSOPT (Claude, only if held-out reviews show
benefit), L-LOAD (Cris, outside reviewer if possible).

**Gate G3 (proposed):** 200 sign-ups in the first six weeks of term; 15% of
week-4 actives paying; p95 non-LLM read under 50 ms at 500 concurrent users;
refunds under 5%; zero privacy incidents; G0 bars still met.

Phase 1 to 3 units get boundaries entries and rpi folders only when their phase
is approved.
