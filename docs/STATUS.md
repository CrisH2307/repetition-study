# Status as of 2026-11-09 evening

Base: `origin/qa` at 8488715 (13:18). A fetch this evening shows no newer commit on
any remote branch. Work done locally after 13:18 and not pushed is unknown.
Sources: code audit of 8488715, planning dossier, and a test run on this branch.

## Build and tests

| Run | Result |
| --- | --- |
| `go build ./...` | Exit 0 |
| `go test ./...` without `DATABASE_URL` (this evening) | 51 PASS, 10 SKIP (Postgres tests), 0 FAIL |
| `go test ./...` with local Postgres 16 (audit, same commit) | 61 PASS, 0 FAIL, 0 SKIP; also green under `-race` |
| `BenchmarkSchedule1000` | 3.4 ms per 1,000 concepts (target 20 ms) |
| Coverage (audit) | checker 92.4%, teach 92.5%, planner 86.2%, srs 83.4%, fakes 83.5%, llm 78.4%, cmd/api 50.0%, cmd/evalcheck 42.1%, cmd/migrate 0% |
| Real model calls | None ever made |

## Units

| Unit | Status | Evidence | Missing |
| --- | --- | --- | --- |
| U0 Setup and contract | Done | `internal/domain/`, `internal/fakes/`, `migrations/0001_core.sql`, `0002_review_events_append_only.sql`, `api/openapi.yaml` (15 paths), `cmd/api` `/healthz`; PRs #1 to #4 | Contract gaps (9, below) |
| U1 Content pack | Partial | `content/source.md` (13 pages, pdftotext), `content/unit.json` (8 concepts, 35 claims, 32 required), `content/allowlist.json` `[]`, `scripts/parse_pdf.py`; `go test ./content/` passes | Cris approval of concepts (D20); formula hand fix (`content/damaged-formulas.md`); `eval/data/`; labels; `scripts/check_labels.py`; `content/seed.sql` |
| U2 Teach engine | Partial | `internal/llm/`, `internal/teach/`; leak suite with fakes; `leak_real_test.go` behind tag `realllm` compiles | Postgres store; any real model run; model ids |
| U3 Checker | Partial | `internal/checker/`, `cmd/evalcheck/`; smoke run with `--model fake` | Any real run; `eval/results/` |
| U4 Scheduler | Done (library) | `internal/srs/`; 7 Postgres tests pass; benchmark 3.4 ms | Event id via contract; never-reviewed concepts not in Due |
| U5 Planner | Done (library) | `internal/planner/`; 8 allocate tests and 2 Postgres tests pass | UTC day windows (F5); hours semantics unconfirmed |
| U6 Digest | Not started | `rpi/U6/` logs empty; no `internal/digest/` | All (D21 proposes superseding) |
| U7 Demo UI | Not started | `rpi/U7/` logs empty; no `web/` | All (D21) |
| GATE | Not started | `rpi/GATE/` logs empty; no `eval/` | Data, keys, labels |
| INT | Not started | `rpi/INT/` logs empty; `cmd/api/main.go:10-17` serves `/healthz` only (0 of 14 `/v1` paths) | Composition root, store, loader, handlers |
| QA | Not started | no `qa/` | All (D21) |
| RELEASE | Not started | no `demo/`, no tag | All (D21) |

Contract gaps recorded by units (need a contract unit, Cris owns): no pack loader
type; Scheduler has no event id parameter; `DueItem` has no reason; Planner has
no GetPlan or PatchBlock; `FurtherPlanning` lacks gaps, citations and change
reasons; `TeachEngine` has no GetReport; no session persistence interface in
domain; `Clock.Now` has no context; cards have no status column.

## Known bugs

| ID | Where | Symptom | Fix owner |
| --- | --- | --- | --- |
| F1 | `internal/teach/engine.go:299` vs `internal/checker/verify.go:35-44` | Checker verifies quotes after whitespace normalization; `finalize` re-checks with raw `strings.Contains`. A quote across a line break is dropped at Finish; label stays contradicted with no quote. Reproduced in the audit | P0-CITE |
| F2 | `internal/teach/student.go:40-42`, `engine.go:133` | Checker-written `probe_hint` reaches the student system prompt; only a 5-word shingle guard. Leak channel for source terms | M-PROBE (D6); measured by P0-LLM and P0-GATE |
| F3 | `internal/teach/store.go`, `engine.go:33` | Only `MemStore`; sessions, turns, claim labels never reach Postgres; sessions lost on restart | P0-STORE |
| F4 | no loader | Nothing loads `content/unit.json` into Postgres; `content/seed.sql` missing | P0-INT |
| F5 | `internal/planner/allocate.go:28,38,281` | Blocks at 09:00 UTC, 04:00 or 05:00 Toronto | P0-TZ |
| F6 | `internal/teach/engine.go:311-316` | Checker outage marks every required claim absent, grades Again, creates gap cards | P0-OUTAGE (D23) |
| F7 | `internal/teach/engine.go:267` | `ApplyTeachBack` gets no event id; retry after restart writes twice | P0-CONTRACT, P0-OUTAGE, P0-SRS |
| F8 | `internal/checker/verify.go:44` | No minimum quote length; one word verifies; locator unchecked | P0-CITE (D22) |
| F9 | `cmd/evalcheck/llm.go`, `internal/llm/gemini.go` | Eval uses its own adapters and prompt; production Gemini has no JSON | P0-LLM |
| F10 | `internal/llm/gemini.go:32`; `internal/teach/student.go:125-150` | MaxTokens 120 with no thinking config [unverified effect]; "e.g. " ends a sentence | P0-LLM; splitter in M-PROBE |
| F11 | `internal/srs/` | Concept-level grade; never-reviewed concepts absent from Due; Again 1d vs Hard 2d | M-CLAIMSRS (D4) |
| F12 | `cmd/migrate/main.go`, `internal/srs/*_test.go` | No advisory lock; srs tests leave rows in the dev DB | P0-SRS |
| F13 | GitHub settings, history | No branch protection; U1 to U5 reached `qa` without PRs | P0-HYG |
| F14 | `cmd/evalcheck/eval.go:87`; `cmd/api/main.go:25`; `internal/llm/anthropic.go:90`; `internal/fakes/fakes.go:211` | Panic on short CSV row; no server timeouts; JSON extraction strips fences only; fake hardcodes `concept-1` | P0-LLM; P0-INT; P0-LLM; P0-CONTRACT |
| T1 | `docs/TESTING.md:108`, `:146` | Secret scan matches its own text; gate file named `gate.md` but evalcheck writes `<model>.md` | P0-HYG |
| C1 | `content/source.md` | Display formulas flattened by pdftotext; quotes of math are garbled | P0-EVALDATA (hand fix of cited regions), M-INGEST |

## Process deviations

- U1 to U5 entered `qa` without PRs: U2 to U5 by local merge 46b47ad, U1 by
  direct commit fecd14b on qa. U1
  landed while its own message said it awaited approval. Regulation 4.
- `main` has an extra empty commit 93c0f89 (Dino's account) not in `qa`.
- Hour 3 gate skipped; U4 and U5 were built on an unproven checker.
- Jev split never run; no manual split logged. Regulation 3.
- UI exception never logged under regulation 10.
- Dino's design note (three dashboards) never written.
- A professor's practice set is tracked in `content/source.md` and
  `content/unit.json`, although the plan said `content/private/` (D7).
- Decisions lived in three places that disagreed; `docs/DECISIONS.md` now replaces them.

## Readiness scores (Claude judgment, 0 to 10, from Fable 5.2)

| Dimension | Score |
| --- | --- |
| Engineering kernel | 7 |
| Closed loop end to end | 1 |
| Core thesis proven (checker on real math) | 0 |
| Content pipeline | 2 |
| Product surface (UI, auth, deploy) | 0 |
| Process discipline | 5 |
| Market validation | 1 |
| Overall | 2 |

## Unverified

- Any work after 13:18 that was not pushed.
- Every real SDK call (Gemini, Anthropic) and the model ids that work with the keys.
- Effect of Gemini thinking tokens on MaxTokens 120.
- Docling quality on this PDF; it was never run.
- OpenAPI validity with a real validator.
- Performance targets other than the scheduler benchmark.
- Copyright position for the committed practice set.
- Market claims in docs/STRATEGY.md marked [unverified].
