# Code dossier: CrisH2307/repetition-study (state of origin/qa at 8488715)

Gathered 2026-11-09 from a full clone at /home/claude/repetition-study (left on detached HEAD = origin/qa). Repo is private on GitHub (gh api: private=true, default branch main, no branch protection on any branch). Nothing was committed, pushed or merged. Postgres tests were run against a local PostgreSQL 16 started under a throwaway user (Docker is not available in this sandbox). Items not verified are marked [unverified].

Citation style: path:line refer to origin/qa.

---------------------------------------------------------------------------

## 0. Executive summary

- qa holds a well-tested set of backend building blocks: domain contract, schema, scheduler (FSRS), planner, checker, teach engine, LLM clients, evalcheck tool, and a draft concept pack. All of it builds, vets and passes tests (61 top-level tests PASS, 0 FAIL, 0 SKIP with Postgres; also green under -race).
- Nothing is wired end to end. cmd/api serves only /healthz (cmd/api/main.go:13-19). 0 of the 14 /v1 endpoints in api/openapi.yaml have a handler. There is no Postgres implementation of teach.Store, no loader that puts content/unit.json into the DB, no subjects/cards/plan-block CRUD, no digest, no UI.
- Units: U0 Done. U4 and U5 Done as libraries (with contract gaps). U1, U2, U3 Partial. U6, U7, GATE, INT, QA, RELEASE Not started (empty rpi files).
- The Hour 3 gate has not been run and cannot be run yet: eval/ does not exist (no explanations.jsonl, no labels), and no real model call (Gemini or Claude) has ever been made by any unit. Every SDK call path is [unverified] against the live APIs.
- Real bug found: a checker-verified citation can be silently dropped at Finish because teach.finalize re-checks with a raw (non-whitespace-normalized) substring while the source text contains newlines (section 8, finding F1). Reproduced with a scratch test.
- Demo-blockers beyond wiring: content/seed.sql (earlier history so a date visibly moves from ~7 days out) was never written; planner places blocks at 09:00 UTC = 04:00 or 05:00 in Toronto; docs/TESTING.md B4 will always show output (self-match), so Dino's "nothing at all" pass criterion cannot be met as written.

---------------------------------------------------------------------------

## 1. Branches

All timestamps 2026-11-09, UTC-5. `ahead` = commits not in origin/qa.

| Branch | Tip | Author | Time | Tip message | ahead of qa | Merged in qa |
|---|---|---|---|---|---|---|
| origin/main | 93c0f89 | Ngoc Vien Do | 12:35 | Initialize repository | 1 | n/a (target) |
| origin/qa | 8488715 | CrisH2307 | 13:18 | Merge branch 'chore/testing-plan' into qa | 0 | n/a |
| origin/chore/testing-plan | ce2bdfd | CrisH2307 | 13:10 | Rewrite test plan in plain language for Dino | 0 | yes |
| origin/feature/setup | d7497cc | CrisH2307 | 11:04 | Add repo scaffold: go.mod with pinned deps, compose, Makefile, rpi stubs | 0 | yes |
| origin/feature/setup-domain | 1273cdf | CrisH2307 | 11:04 | Add domain types, interfaces and deterministic fakes | 0 | yes |
| origin/feature/setup-db | e7111e0 | CrisH2307 | 11:04 | Add schema migrations, append-only review_events trigger and runner | 0 | yes |
| origin/feature/setup-api | 3707359 | CrisH2307 | 11:04 | Add OpenAPI contract and /healthz | 0 | yes |
| origin/feature/teach-engine | edcd055 | CrisH2307 | 11:23 | Add 10-probe leak suite with a real-model variant behind build tag realllm | 0 | yes |
| origin/feature/checker | 213ea9f | CrisH2307 | 11:21 | Add evalcheck to report checker gate numbers per model | 0 | yes |
| origin/feature/scheduler | ad3f942 | CrisH2307 | 11:20 | Record U4 research, plan and log | 0 | yes |
| origin/feature/planner | c9b9801 | CrisH2307 | 11:25 | Record U5 research, plan and implementation log | 0 | yes |
| origin/feature/content-pack | fecd14b | CrisH2307 | 12:44 | Add U1 parser, source text and draft concept pack | 0 | yes |

`git branch -r --merged origin/qa` lists every branch except origin/main (and HEAD). Because every feature branch is an ancestor of qa, `git diff qa..feature/x` is empty for all of them; their individual contributions are:

- feature/setup (U0 1/4): .env.example, Makefile, compose.yaml, go.mod, go.sum, internal/deps/deps.go (build tag `tools`, pins go-fsrs/v4, pgx/v5, genai, anthropic-sdk-go), 40 empty/placeholder rpi files. Kit commit 3c4edb3 ("Add Claude Code project kit") adds CLAUDE.md, .claude/*, docs/, scripts/jev_split.py, scripts/unit.sh, rpi/*/PROMPT.md.
- feature/setup-domain (U0 2/4): internal/domain/{types,interfaces}.go, internal/fakes/{fakes,fakes_test}.go.
- feature/setup-db (U0 3/4): migrations/0001_core.sql, 0002_review_events_append_only.sql, embed.go, migrations_test.go, cmd/migrate/main.go, scripts/seed.sql.
- feature/setup-api (U0 4/4): api/openapi.yaml, cmd/api/main.go, cmd/api/main_test.go.
- feature/teach-engine (U2): internal/llm/{anthropic,gemini,retry,llm_test}.go, internal/teach/{engine,student,store}.go + tests (engine_test, leak_test, leak_real_test with build tag realllm).
- feature/checker (U3): internal/checker/{checker,verify}.go + test, cmd/evalcheck/{main,eval,llm}.go + test.
- feature/scheduler (U4): internal/srs/{logic,scheduler}.go + tests.
- feature/planner (U5): internal/planner/{allocate,store,progress}.go + tests.
- feature/content-pack (U1): content/{allowlist.json,damaged-formulas.md,source.md,unit.json,unit_test.go}, scripts/parse_pdf.py, rpi/U1/*.
- chore/testing-plan: docs/TESTING.md (two commits: e2d76ab "Add test plan for Dino", ce2bdfd "Rewrite ... plain language").

File boundary audit (.claude/boundaries.json): every branch only touched paths allowed for its unit (checked with git diff --name-only per branch range). No branch edited internal/domain, api/ or migrations/ after U0.

### 1.1 Process observations (history shape)

- qa history uses local merges: `4bae930` merges setup-domain/setup-db/setup-api (octopus), `46b47ad` merges teach-engine/checker/scheduler/planner (4-way octopus), `8488715` merges chore/testing-plan. Only PRs #1-#4 (U0) and #6 (test plan) exist on GitHub, all "closed, merged" (PRs 1-4 at the same instant 16:15:35Z). There are no PRs for U1, U2, U3, U4, U5 (gh api pulls/5 and /7 return 404). So CLAUDE.md "Small PRs into qa, one function each" was not followed for U1-U5; they entered qa by local merge/fast-forward (fecd14b and e2d76ab sit directly on qa's first-parent line). The guard hook (.claude/hooks/guard.py) only constrains Claude sessions, so this was possible. No branch protection is configured on GitHub.
- Duplicate commit: 25ba649 and b4c3cb8 are both "Add pure planner allocator with deterministic greedy 30 minute blocks" on feature/planner (impl-log also has the same line twice).

### 1.2 main versus qa: the root question

Verified. origin/main does NOT have a separate root.
- `git merge-base origin/main origin/qa` = 98ddad8 (Cris's "Initialize repository", 10:54 -0500). `git rev-list --max-parents=0 origin/main origin/qa` returns only 98ddad8.
- origin/main = 93c0f89 (Ngoc Vien Do, 12:35) whose parent is 98ddad8. 98ddad8 and 93c0f89 both have the empty tree (4b825dc...). So main is "root + one empty commit by Ngoc Vien Do"; the commit adds no files.
- `git diff origin/main origin/qa` = 112 files, +8762 lines (everything). `git merge-tree --write-tree origin/main origin/qa` is clean (no conflicts), tree c172da5.
- Consequence for a qa -> main PR: it is a normal PR. GitHub will compute merge-base 98ddad8, show all 112 files / 24 qa commits, and be mergeable without conflicts. A merge commit will be created on main (not a fast-forward because main has the extra commit 93c0f89); squash would also work. There is no unrelated-histories problem. The only wrinkle is that qa lacks 93c0f89, so the first merge pulls it in as an empty commit. Also the "Initialize repository" instruction in docs/CLAUDE-CODE-SETUP.md (`git commit --allow-empty ... && git push -u origin main`) is the cause of the two init commits (Cris locally, then another from Ngoc Vien Do's account).
- The guard hook blocks `git push ... main` and `gh pr merge`, so the release flow in rpi/RELEASE/PROMPT.md (PR qa -> main, Cris merges, tag demo-v1) is the only intended path. No tag exists yet.

---------------------------------------------------------------------------

## 2. File tree of origin/qa (112 files)

```
.claude/{boundaries.json,settings.json,hooks/guard.py,skills/{handoff,status,unit}/SKILL.md}
.env.example  .gitignore  CLAUDE.md  Makefile  compose.yaml  go.mod  go.sum
api/openapi.yaml
cmd/api/{main.go,main_test.go}
cmd/evalcheck/{eval.go,eval_test.go,llm.go,main.go}
cmd/migrate/main.go
content/{allowlist.json,damaged-formulas.md,source.md,unit.json,unit_test.go}
docs/{CLAUDE-CODE-SETUP.md,PLAN.md,TESTING.md}
internal/checker/{checker.go,verify.go,checker_test.go}
internal/deps/deps.go
internal/domain/{interfaces.go,types.go}
internal/fakes/{fakes.go,fakes_test.go}
internal/llm/{anthropic.go,gemini.go,retry.go,llm_test.go}
internal/planner/{allocate.go,progress.go,store.go,allocate_test.go,store_test.go}
internal/srs/{logic.go,scheduler.go,logic_test.go,scheduler_test.go}
internal/teach/{engine.go,student.go,store.go,engine_test.go,leak_test.go,leak_real_test.go}
migrations/{0001_core.sql,0002_review_events_append_only.sql,embed.go,migrations_test.go}
rpi/{U0..U7,GATE,INT,QA,RELEASE}/{PROMPT,research,plan,impl-log}.md
scripts/{jev_split.py,parse_pdf.py,seed.sql,unit.sh}
```
Absent directories: eval/ (entire), web/, internal/digest/, internal/app/, qa/, demo/. Absent files named in prompts: eval/data/*, scripts/check_labels.py, content/seed.sql, rpi/work-split.md, rpi/U*/HANDOFF.md. Source is ~8.3k lines in total including docs and the 14 KB source text.

---------------------------------------------------------------------------

## 3. Package by package (implemented, stubbed, missing)

### api/openapi.yaml (335 lines, OpenAPI 3.0.3)
15 paths: /healthz and 14 under /v1 (subjects GET/POST; plan/generate; plan GET; plan/blocks/{id} PATCH; teach/sessions POST; turns (SSE); finish; report GET; reviews/due; cards/{id}/review; cards/{id} PATCH; concepts/{id}/override; analytics/further-planning; digest/today). U0 impl-log: "15 paths, 24 schemas, no broken $refs, linted with a local script only" [unverified with a real OpenAPI validator]. Implemented handlers: /healthz only.

### cmd/api (stub)
main.go:13-19 `newMux()` registers only `GET /healthz`. main.go:22-30 starts `http.ListenAndServe`. No DB pool, no LLM clients, no wiring, no logging middleware, no CORS (UI will need it or a proxy), no graceful shutdown. Test: TestHealthz.

### cmd/migrate (implemented)
Applies embedded migrations/*.sql in name order in transactions with a schema_migrations table. Comment admits no lock (concurrent runs can race). Idempotent: second run prints nothing.

### cmd/evalcheck (implemented, never run against a real model)
main.go: flags --model, --unit, --explanations, --labels, --out, --price-in, --price-out; writes eval/results/<model>.md. eval.go: loaders, `worst()` collapse, `stats.report()` (gate numbers, confusion table, latency, tokens, optional cost). llm.go: its own provider adapters (fake, gemini via genai, claude via anthropic SDK) with token metering. Gaps: no retry/timeouts in these adapters (llm.go:75 `anthropic.NewClient` default SDK retries, no per-call timeout); it does NOT use internal/llm, so the eval prompt differs from production (internal/llm/anthropic.go appends "Reply with one JSON value and nothing else." to the system prompt, evalcheck/llm.go:134 does not); `loadLabels` indexes `r[lblCol]` without a length check (eval.go:87, panics on a short CSV row); "cost per session" asked for by rpi/GATE is not computed (cost per call only); it reports explanation-level worst label, so the gate metric "wrong flagged contradicted or absent" counts gold labels contradicted OR absent as "wrong" (eval.go:143), and partial is excluded.

### internal/domain (implemented, frozen)
types.go: ClaimLabel, Grade, EventType, SourceKind, Level, PlanKind, Source, Subject, Concept, Claim, TaughtClaim, Citation, ClaimResult, ReviewEvent, PlanBlock, Card, Message, LLMRequest, Delta, TeachSession, Report, CheckInput, DueItem, Plan, ConceptProgress, SubjectProgress, FurtherPlanning, DigestItem, Digest, DemoUserID const. interfaces.go: LLM, TeachEngine, Checker, Scheduler, Planner, Digester, Clock. Contract gaps recorded by units (not fixed): no Unit/loader type (U1 invented `{subject,sources,concepts}`); Scheduler has no event-id param (srs uses ctx value `WithEventID`); DueItem has no reason (srs has `DueWithReasons`); Planner has no GetPlan/PatchBlock; FurtherPlanning domain type lacks gaps, citations, change reasons, recall-vs-retrievability (planner.Progress returns the richer struct); TeachEngine has no GetReport; no session persistence interface (teach.Store invented by U2); Clock.Now has no context; Card has no status column or schedule.

### internal/fakes (implemented)
Deterministic fakes of every interface (fakes.go: Clock, LLM, TeachEngine, Checker, Scheduler with fixed intervals Again 1d / Hard 3d / Good 7d / Easy 14d, Planner, Digester). Tests: determinism, wrong explanation pulls date earlier. Note fakes.Scheduler.ReviewCard hardcodes concept "concept-1".

### internal/llm (implemented against stub servers only)
- retry.go: `Retry.do` with attempts 3, backoff 300ms doubling, 30s per-attempt timeout; `retryable()` matches error text for 429/5xx/UNAVAILABLE etc. (string match, flagged as ponytail).
- gemini.go: `Gemini.Stream` (genai GenerateContentStream; retries only before first text; terminal Delta.Err after). `JSON` returns ErrUnsupported. `MaxTokens` default 120 (gemini.go:32). Risk [unverified]: if STUDENT_MODEL is a Gemini "thinking" model, thinking tokens may count against MaxOutputTokens=120 and produce empty or truncated replies; no ThinkingConfig is set. If the checker bake-off picks a Gemini tier, production has no Gemini JSON implementation (only evalcheck has one).
- anthropic.go: `Anthropic.JSON` (system prompt + user msgs, extracts JSON by stripping a markdown fence only, json.Unmarshal; decode failure not retried). `Stream` returns ErrUnsupported. MaxTokens 2048 (large claim lists with citations could be truncated, which would yield invalid JSON [unverified]).
- Env: GEMINI_API_KEY, STUDENT_MODEL, ANTHROPIC_API_KEY, CHECKER_MODEL. STUDENT_MODEL and CHECKER_MODEL are NOT in .env.example (U2 research notes it).
- Tests: retry, Anthropic JSON and Gemini stream against httptest servers (llm_test.go:51,72). No real call ever made.

### internal/checker (implemented, unit tested, 92.4% coverage)
- checker.go: `Checker{LLM,Model,AllowAbsent,Log}`; `Check`/`CheckDetailed`; strict system prompt (checker.go:29-41); transcript, taught text and latex fenced as untrusted with a close-tag defanging helper (`untrusted`, :44); only approved sources are sent.
- verify.go: `normalize` (whitespace collapse), `verifyCitation` (exact substring after whitespace normalization, no case folding; approved source id required; empty quote rejected; NO minimum quote length, so a one-word quote verifies), `hintLeaks` (5-word shingle, hintRun=5 at :13), `approved` (course always; allowlist only with ReviewedBy and ReviewedAt), `enforce` (drops invalid labels; drops `absent` unless AllowAbsent; unknown claim id becomes unmatched; removes any correction lacking a verified citation and downgrades contradicted to partial at :155; removes leaking hints; in finish mode appends `absent` for unseen required claims).
- Not checked by code: the citation `locator` is model-supplied and unverified; that the quote actually relates to the claim; that the correction text is faithful to the quote.
- Tests: correct quote, paraphrase, case change, unapproved source, quote from wrong source, injected transcript, hint leak, absent rules, invalid label, reviewed allowlist.

### internal/teach (implemented against MemStore only)
- student.go: `StudentContext{ConceptName, Taught, Probe}` (no source fields; locked by reflection test), prompt rules (:21-29), untrusted-speech tags with sanitization, `pickProbe` (contradicted hint, then partial hint, then generic follow-up; skips used hints and hints sharing a 5-word run with source or claim text), `sentenceCap` (2 sentences).
- engine.go: `Engine.Start/Turn/Finish`. Turn saves the turn, starts an async checker goroutine with timeout (default 20s, engine.go:19), streams the student reply via domain.LLM, logs TTFT. Finish waits up to FinishWait=8s (:20), uses the latest completed check, `finalize`s, saves, calls `Scheduler.ApplyTeachBack` once (in-memory guard via s.report), caches the report.
- store.go: `Store` interface + `MemStore`. NO Postgres implementation exists (grep for SaveResults/LoadConcept finds only the interface, MemStore and engine). Consequences: sessions, turns, claim_labels are never written to DB in any real run; planner.loadExplanation (which reads claim_labels) therefore returns no data in a real run; GET /teach/sessions/{id}/report has no backing store; Engine sessions live only in a map (engine.go:33), lost on restart and never evicted.
- Tests (92.5% coverage): context fields, source text never reaches student, probe order, 2-sentence cap, TurnDoesNotWaitForChecker, checker failure/panic, Finish fills absent + verifies citation + schedules once, Finish stops waiting, untrusted block cannot be closed, TTFT logged, empty turn, 10-probe leak suite (fakes only) + realllm-tagged real-model test (compiled, never run).

### internal/srs (implemented; Postgres tests pass)
- logic.go: `newFSRS` (go-fsrs v4.0.0 default weights, retention 0.9, `EnableShortTerm=false` so a new card gets Again 1d / Hard 2d / Good 3d; EnableFuzz is default false in go-fsrs [read from module, default value not re-verified]), `teachBackGrade` (worst of contradicted/absent=Again, partial=Hard, else Good, never Easy; skips non-required claims; error when nothing countable), `interleave`.
- scheduler.go: `Scheduler.ApplyTeachBack` (:202), `ReviewCard` (:171), `Override` (:182; due date only, reason required), `Due`/`DueWithReasons` (:334), `gapCards` (:257; one card per non-supported result, deduped by claim id or front text, "open" = "exists"), per-concept advisory lock (:54), idempotency via `WithEventID` ctx value, uses Clock. Only INSERT into review_events anywhere (grep for UPDATE/DELETE review_events in non-test Go: none).
- Benchmark: BenchmarkSchedule1000 = 3.4 ms/op on this box (target 20 ms).
- Gaps: a concept only gets a concept_schedules row after its first teach-back, card review or override, so `Due` (JOIN concept_schedules, scheduler.go:334+) never lists never-reviewed concepts; ReviewCard moves the concept schedule (cards carry no FSRS state, contrary to U4 PROMPT wording "Cards under it have their own FSRS state", documented in research.md); ApplyTeachBack called by Finish passes no event id so it is not idempotent at DB level (guard is only in-memory); unmatched statements (ClaimID nil) labeled partial/contradicted count toward the grade; no `PATCH /cards/{id}` implementation (grep for UPDATE cards: none).

### internal/planner (implemented; Postgres tests pass)
- allocate.go: pure `Config.Allocate` (greedy 30-minute blocks, score = weight * level * urgency * max(0, target - mean projected retrievability), division by (1+assigned), ties by subject id, 30% new share, locked blocks respected, teach before review, teach block >= 24h ahead, plain-English reasons). Config holds all constants (DefaultConfig :34).
- store.go: `Store.Generate` (:56; replaces the user's unlocked future blocks of the week in one tx, loads subjects/concepts/events with parameterized SQL), `FurtherPlanning` maps from `Progress`. progress.go: `Progress` returns recall (share of card reviews Good/Easy) and explanation (share of required claims supported in latest teach-back) separately, gaps with citations, this-week vs next-week hours and reason string.
- Issues: slots start at 09:00 UTC (Config.DayStartHour=9, allocate.go:28,281) = 04:00 EST or 05:00 EDT Toronto; 8 slots a day gives 4 hours a day max in UTC mornings. weekly_hours is only a ceiling and demand scales with retention gap (`Demand = ceil(ceiling * min(1, gap/0.5))`, allocate.go:154), so a subject with decent retention gets well under its weekly_hours (design proposal flagged in U5 research). `place` returns an error ("no free slot left", :301) rather than degrading when the week has too few free slots (e.g. teach lead time pushes out late in the week). weekStart is not validated as a Monday. FurtherPlanning drops gaps, citations, change reasons (domain has no room).
- Nothing creates concepts: the seeded demo subject (scripts/seed.sql) has no concepts, so Generate yields only "No concepts yet" `new` blocks until a loader populates sources/concepts/claims from content/unit.json (no such loader exists).

### internal/deps (build-tag stub)
deps.go pins four modules under tag `tools`.

### migrations (implemented)
0001_core.sql (149 lines): users, subjects, sources, concepts (UNIQUE subject_id,name), claims, teach_sessions, teach_turns, taught_claims, claim_labels (CHECK correction requires citation_source_id and citation_quote), cards, concept_schedules (one FSRS state per concept), review_events (+index), plan_blocks, digests (UNIQUE user_id,digest_date). 0002: trigger `review_events_reject_change` for BEFORE UPDATE OR DELETE (row level) and BEFORE TRUNCATE (statement level), error text "review_events is append-only". Verified live: `TRUNCATE review_events;` fails with "review_events is append-only (TRUNCATE rejected)". A DB superuser can still DROP the trigger. The test suite leaves rows behind in the dev DB (append-only prevents cleanup; my local DB had 102 rows after the run).

### content (draft)
- unit.json (356 lines, valid JSON, keys subject/sources/concepts): subject MATH 6602 (id ...00a1, weekly_hours 4, intermediate, exam 2026-12-15), one course source (id ...00b1, 14,310 chars), 8 concepts, 35 claims (32 required): Exponential races 4/3; Superposition of Poisson processes 4/4; Poisson counts and waiting times 4/4; Generators and transition matrices 4/4; Stationary distribution from balance equations 5/3; Sojourn and hitting times 4/4; Birth-death recurrence criteria 5/5; Long-run behaviour of birth-death queues 5/5 (required/total shown as total/required). Each claim has source_id and a locator like "p.5, Q1".
- `go test ./content/` passes (strict decode, uuid ids, locators resolve to headings on the cited page, source text equals normalized source.md, allowlist empty, 3 to 5 required claims per concept, at most 8 concepts).
- allowlist.json = `[]` (by design, licenses not reviewed). The test pins the entry fields.
- source.md (670 lines): pdftotext -layout output with `<!-- page N -->` markers, 13 pages. Math is still damaged (the hand-fix step listed in content/damaged-formulas.md was not done, source.md page 2 still shows split fractions). Consequence: any citation quote of a formula is a quote of garbled text, and exact-substring verification will be brittle for math.
- Claims are largely worked-example facts ("P(N(1) >= 10) = ... = 0.0839", specific generator matrices and stationary vectors) rather than concept statements; this will make teach-back about recalling numbers from a practice-set solution. Design risk, not a bug.
- Mismatch with docs/PLAN.md: the PLAN calls the demo unit "MATH 6602 Chapter 2, Markov chains"; the actual file is Practice Problem Set 3 (continuous-time Markov chains, Poisson, birth-death) (rpi/U1/research.md). PLAN open question on which unit and rights to use is still unchecked ("[ ]"). Full text of a professor's practice set is committed to a private repo (source.md and unit.json) [copyright/permission position not researched, as PLAN itself admits].
- scripts/parse_pdf.py: Docling first (unverified API, never run) then pdftotext; `--sync` copies normalized source.md into unit.json.

### eval/ (missing)
No eval/ directory at all. Missing: eval/data/explanations.jsonl (20 explanations), labels_cris.csv, labels_dino.csv, labels_final.csv, scripts/check_labels.py, eval/results/*. So the labels by Cris and Dino do not exist (not even empty templates). evalcheck expects `id`/`concept_id`/`text` JSONL and `id`/`label` CSV (shapes assumed by U3, [unverified] against U1 since U1 never produced them).

### internal/digest (missing), web/ (missing), internal/app/ (missing), qa/ (missing), demo/ (missing).
`digests` table exists in 0001 but nothing reads or writes it.

---------------------------------------------------------------------------

## 4. Unit status against .claude/boundaries.json and docs/PLAN.md

| Unit | Branch | Status | Evidence |
|---|---|---|---|
| U0 Setup | feature/setup (+3 sub-branches) | Done | go.mod pinned, compose.yaml, Makefile (up/migrate/test/run/seed), migrations + trigger test (fails with trigger dropped per impl-log), domain, fakes, openapi, /healthz. Report items not delivered: contract-wrongness list lives only in plan.md ("users table, Card type, Clock has no ctx"). |
| U1 Concept pack | feature/content-pack | Partial | Done: parse_pdf.py, source.md, damaged-formulas.md, unit.json draft, allowlist.json. Not done: Cris approval of concept list/demo concept (plan.md step 2 "STOP"), eval/data/explanations.jsonl, labels_cris/dino.csv, check_labels.py, content/seed.sql, hand-fix of formulas, labels_final.csv. Done-condition ("labels_final.csv complete") not met. |
| U2 Teach engine | feature/teach-engine | Partial | Done: llm clients, engine, student context, leak suite with fakes. Not done: Postgres Store (INT), any real Gemini/Claude run, real-model leak run, model ids unknown. |
| U3 Checker | feature/checker | Partial | Done: checker + verification + evalcheck + tests. Not done: any run (no keys, no labels), eval/results/*. "Run for at least one Gemini and one Claude tier" not done. |
| U4 Scheduler | feature/scheduler | Done (library) | All prompt tests present and passing with Postgres (mappings, 7-day pull earlier, override stability unchanged, retry writes once, benchmark 3.4 ms). Contract deviations documented. |
| U5 Planner | feature/planner | Done (library) | All prompt tests present (determinism, hours, locked blocks, worse retention raises hours) plus Postgres tests; FurtherPlanning contract deviations documented. |
| U6 Digest | feature/digest | Not started | No internal/digest, rpi/U6 files empty (only PROMPT.md). |
| U7 Demo UI | feature/demo-ui | Not started | No web/, rpi/U7 empty. |
| GATE | feature/gate-results | Not started | rpi/GATE empty; blocked on eval data + keys. |
| INT | feature/integration | Not started | rpi/INT empty; cmd/api only /healthz; no internal/app. |
| QA | chore/qa-run | Not started | No qa/; docs/TESTING.md (human test plan for Dino) exists but is not qa/report.md. |
| RELEASE | chore/release | Not started | No demo/RUNBOOK.md, no tag. |

PLAN.md acceptance items status: add 2 subjects + weekly plan (planner library only, no API/UI: not met); voice explanation with planted mistake yields cited gap (engine + checker libraries only, no real model: not met); next review date moves earlier (srs DB test proves it: met at library level); gap creates editable card (srs gapCards yes; edit endpoint no); Further Planning shows recall and explanation separately (planner.Progress yes, no API/UI); digest (not met); manual override saved and logged (srs yes, no API); full demo run on qa (not met).

---------------------------------------------------------------------------

## 5. Documents read (summaries)

### CLAUDE.md (42 lines)
Hard rules: code owns state; AI student never receives source material, only the human's claims plus one probe target; student only asks questions; checker sees sources and every correction needs a code-verified exact-substring citation; approved sources = course unit + reviewed allowlist; student speech is untrusted data; sources hidden during explanation; review_events append-only; no claims of learning gains. Stack: Go (net/http, pgx, SSE), Postgres, go-fsrs, Gemini student, Claude checker, Python only in scripts/parse_pdf.py, React/Vite/TS/KaTeX UI. Working rules: verify, stay in the file boundary, never commit to main, small PRs into qa, keep rpi files current, no secrets, no em dashes, stop at 50% context, final report < 200 words. Deadline: 5pm America/Toronto 2026-11-09.

### docs/PLAN.md (237 lines)
Verdict "accept with revisions, pending the Hour 3 checker test". Closed loop: teach-back finds gaps, gaps change FSRS dates, dates change plan. Confirmed decisions table (concept as review unit, FSRS with manual override, voice-first, one daily digest, Go + Postgres, closed loop on one pre-loaded course). Gate bars: wrong flagged >= 4 of 5; correct falsely contradicted <= 1 of 10; 100% corrections cited; 0 untaught facts in 10 probes. Performance targets (non-LLM p95 < 50 ms at 50 users; recompute < 20 ms; first token < 1.5 s; checker < 6 s). Cut order if late: digest delivery, outside-source pack, plan editing, generated cards (note: .claude/skills/status/SKILL.md lists a slightly different cut order: digest delivery, plan editing, generated cards, and omits outside-source pack). Open questions: demo unit and rights unresolved; outside-source pack named (Leetcode etc.) but allowlist is empty; Jev key question answered ("already on Cris's computer"); one design-research artifact (Dino's 30-minute dashboards note) not done.

### docs/TESTING.md (203 lines): Dino's plain-language test plan
Part A setup (Mac, Docker, Go 1.26+, GitHub Desktop; `make up migrate test`), Part B safety checks (make test, `go test -race ./...`, TRUNCATE must fail with "review_events is append-only", secret grep, idempotent migrate, `go test ./internal/teach/...`), Part C the Hour 3 gate (20 explanations labelled independently, then reconciled to labels_final.csv; report at eval/results/gate.md), Part D full demo run D1-D10, Part E stage risks, Part F reporting format. Defects found in the plan:
- B4 command `git grep -n -e AIza -e sk-ant -e sk-or $(git rev-list --all)` matches the text of docs/TESTING.md itself in every commit that contains it (I ran it: hits at TESTING.md:108 in 8488715 and ce2bdfd, and a table row at :38 in fecd14b/e2d76ab). Expected result "nothing at all" can never be reached. Real secret scan found no key material in history (the only matches are those self-references; no .env ever committed).
- B6 and Part C depend on files that do not exist yet (eval/results/gate.md, labels_final.csv).
- Part C says Cris runs the checker and Dino reads gate.md, but gate.md is produced by the GATE unit, which has not started; evalcheck writes `<model>.md`, not `gate.md`.
- Windows users are told to ask Cris first (Makefile uses POSIX shell constructs).

### docs/CLAUDE-CODE-SETUP.md (30 lines)
First-time shell commands, `scripts/unit.sh <unit>` per unit worktree, `/status`, `/handoff`, edit PROMPT.md to change tasks, Jev split via `python3 scripts/jev_split.py` (writes rpi/work-split.md, which does not exist: Jev split not run, PLAN regulation 3 unmet).

### .claude/skills
- unit: runs a unit from rpi/$0/PROMPT.md using Research, Plan, Implement; stop after two failed fixes; PRs via `gh pr create --base qa`.
- status: report progress to the 5pm deadline with a table and recommend cuts (disable-model-invocation).
- handoff: writes rpi/$0/HANDOFF.md when context passes 50%.

### .claude/hooks/guard.py (43 lines) and settings.json
PreToolUse hook on Bash and Edit/Write. Blocks: `git push ... main`, `git commit|merge` while on main or qa, `gh pr merge`, `git add .env`; enforces per-unit allow/deny paths from boundaries.json by current branch. Weaknesses: regex-based (a `git push origin HEAD:main` is caught by `\bmain\b`, but `git push origin feature/main-fix` is a false positive; shell tricks such as `git -C dir commit` still match, `git -c ... merge` too; piping through `sh -c` or scripts escapes it); it keys the unit off the current branch, so on a non-unit branch (e.g. chore/testing-plan, fix/*) there is no path restriction; the hook was evidently not active for the local merges into qa. settings.json allows `git push *` and `git add *` broadly; denies reading .env.

### scripts
- unit.sh: creates sibling worktree `../sr-<branch>` from origin/qa and launches `claude "/unit <U>"`.
- jev_split.py: scores units via OpenRouter (model typesafe/jev-1.13), key from env or ~/.config/jev/openrouter_key; writes rpi/work-split.md. Not run (no work-split.md). Its UNITS list omits U0, U6 detail, GATE, RELEASE.
- seed.sql: seeds demo user (id ...0001) and subject MATH 6602 (id ...00a1, 4 hours, intermediate, exam 2026-12-15). Idempotent. No concepts, no teach-back history.
- parse_pdf.py: see content.

### rpi per-unit documents (verbatim where it matters)
PROMPT.md exists for all 12 units. research/plan/impl-log are non-empty only for U0-U5; for U6, U7, GATE, INT, QA, RELEASE they are 0 bytes.

U0 impl-log (6 lines): "Full `make up migrate test` green. PRs are split into 4 branches". "Trigger test: passes, and fails with the update/delete trigger dropped (mutation check)." "api/openapi.yaml (15 paths, 24 schemas, no broken $refs, linted with a local script only)". plan.md risk: "contract gaps are listed in the final report (users table, Card type, Clock has no ctx)". research: go-fsrs v4.0.0 (FSRS v6 weights, 35 numbers), pgx v5.12.0, genai v1.77.0, anthropic-sdk-go v1.84.0, "Unverified: SDK call signatures (U2 and U3 check them)".

U1 impl-log: "Parser written, run once with pdftotext fallback: 13 pages -> content/source.md." "unit.json draft: 8 concepts, 35 claims (32 required)." "allowlist.json = [] ; test pins its fields." plan.md: "STOP: Cris approves list and demo concept." Steps 4 to 6 (explanations, labels, seed.sql) "After approval", not done. research: "Mismatch with docs/PLAN.md: this is NOT Chapter 2 discrete-time Markov chains." "Docling ... API is UNVERIFIED here: never imported or run." "Unverified: whether Docling gives better math than pdftotext on this file." Q2 omitted because of a typo in the PDF.

U2 impl-log: "llm: retry helper, Gemini stream, Anthropic JSON built; tests pass against httptest stubs (real API unverified)". "go test -race ./internal/teach passes (fakes)". "leak suite: 10 probes with fakes (leak_test.go) and realllm-tagged real test (compiled only, unverified)". plan.md risks: "real SDK calls unverified; no pg Store; checker results lag one turn; probe leak guard is a 5-word shingle check, not semantic." research: "No live call made: no keys." "STUDENT_MODEL and CHECKER_MODEL are not in .env.example ... Unverified: model ids that work with the keys." "INT must supply" a pgx Store.

U3 impl-log: "Not run: real Gemini and Claude tiers (no keys in .env, labels_final.csv not merged). Real SDK calls unverified." "smoke run with --model fake on temp data ok; gemini without key errors cleanly." research Unverified list: shapes of explanations.jsonl, labels_final.csv, unit.json; "Hint overlap threshold (5 consecutive words) and no minimum quote length are my choices, not in the prompt." "Prices: none hardcoded."

U4 impl-log: "Added pure logic ... unit tests and benchmark pass, 1,000 concepts about 2-3 ms." "DB tests pass, also with -race and twice in a row." research: default FSRS params use minute learning steps so `EnableShortTerm=false` chosen ("Unverified against the FSRS authors' intent"); contract frozen so event id rides on context and reasons come from `DueWithReasons`; "cards has no status column, so 'open gap card' means 'a card exists for that claim'". plan risks: "results with no claim id count toward the grade; same-instant events tie when picking the latest event for a due reason; test rows stay in the dev DB (append-only table)."

U5 impl-log (5 lines, one duplicated): "allocate.go + tests: 8 pass" (twice), "store.go: Generate replaces unlocked future blocks in one tx ... Progress and FurtherPlanning in progress.go", "store_test.go: 2 DB tests in a rolled-back tx, pass against local Postgres". research: "Unverified: exact hours semantics ... Read as: weekly_hours is a ceiling, hours demanded = ceiling * min(1, gap/FullGap). Proposal, in Config." "Contract gap: ConceptProgress has one Recall field and SubjectProgress has no change reasons or gaps." plan risks: "hours semantics, UTC day windows, current-week lead time."

U6, U7, GATE, INT, QA, RELEASE PROMPT.md: tasks defined (digest with Toronto day boundary + suppression; React/Vite/KaTeX UI with six tabs and Playwright; gate comparison against the four bars with no prompt tuning; integration replacing fakes in order scheduler, planner, checker, teach engine, digest, UI plus one full-loop test; QA load tests and security checks, report bugs not fix; release runbook and PR qa -> main then tag demo-v1). Note: U7 prompt says "Use Dino's design note if it exists" (PLAN says it does not).

---------------------------------------------------------------------------

## 6. Build, vet, test results (origin/qa, Go 1.26.3, Linux amd64)

Environment: Docker daemon unavailable (`docker compose up` impossible). Used local PostgreSQL 16 started by a throwaway user (user repetition, db repetition, port 5432, trust auth), applied `go run ./cmd/migrate` ("applied 0001_core.sql", "applied 0002_review_events_append_only.sql"). DATABASE_URL=postgres://repetition@localhost:5432/repetition?sslmode=disable. Postgres was stopped afterwards.

- `go build ./...`: exit 0, no output.
- `go vet ./...`: exit 0. `go vet -tags realllm ./...`: exit 0 (realllm test file compiles).
- `go test ./...` (with DB): all ok.
```
ok cmd/api 0.004s | ok cmd/evalcheck 0.006s | ? cmd/migrate [no test files] | ok content 0.003s
ok internal/checker | ? internal/domain [no test files] | ok internal/fakes | ok internal/llm 0.056s
ok internal/planner 0.035s | ok internal/srs 0.073s | ok internal/teach 0.076s | ok migrations 0.007s
```
- `go test -race -count=1 ./...`: all ok (same packages; internal/srs 1.148s, internal/teach 1.094s).
- Verbose count: 61 top-level `--- PASS`, 0 FAIL, 0 SKIP with the database present. The subtests/DB tests (srs 7, migrations 1, planner 2) did run.
- Without DATABASE_URL: 8 tests SKIP (srs: TestApplyTeachBackGradeMappings, TestContradictedTeachBackPullsReviewEarlier, TestOverrideLeavesMemoryAlone, TestRetriedEventWritesOnce, TestGapCards, TestDueReasonsAndInterleave, TestReviewCardMovesConceptSchedule; migrations: TestReviewEventsIsAppendOnly). The planner DB tests did not skip in that run only because they fall back to the default localhost URL, which was up; they Skipf if no DB (store_test.go:26,36).
- Benchmark: `BenchmarkSchedule1000-2  200  3403058 ns/op` (3.4 ms, target 20 ms).
- Coverage: cmd/api 50.0%, cmd/evalcheck 42.1%, cmd/migrate 0.0%, checker 92.4%, fakes 83.5%, llm 78.4%, planner 86.2%, srs 83.4%, teach 92.5%, content/migrations n/a. internal/domain no tests.
- Not run / impossible here: `make up` (Docker), any `-tags realllm` test, any real-model evalcheck, Playwright, load test, `docker compose exec ... TRUNCATE` (instead ran the same TRUNCATE via psql: rejected as expected).
- content/unit.json validity: valid JSON, `go test ./content/` passes (see section 3). eval/data labels: do not exist (no eval/ directory), so no labels by Cris or Dino have been filled; not even empty templates.
- Secret scan over all commits (`git grep -e AIza -e sk-ant -e sk-or-`): only self-references in docs/TESTING.md; no key found; .env never committed (`git log --all -- .env` empty). `.env.example` has names only.
- SQL injection scan: all SQL in internal/ and cmd/ uses `$n` parameters; no string-built queries (grep for Sprintf/concatenated SELECT/INSERT found none).
- Em dash scan: 0 occurrences in the repo (CLAUDE.md rule met).

---------------------------------------------------------------------------

## 7. Contract check

1. OpenAPI vs handlers: of 15 paths only `GET /healthz` is served. Missing handlers (all 14): GET/POST /v1/subjects; POST /v1/plan/generate; GET /v1/plan; PATCH /v1/plan/blocks/{id}; POST /v1/teach/sessions; POST /v1/teach/sessions/{id}/turns (SSE with events token/done/error); POST .../finish; GET .../report; GET /v1/reviews/due; POST /v1/cards/{id}/review; PATCH /v1/cards/{id}; POST /v1/concepts/{id}/override; GET /v1/analytics/further-planning; GET /v1/digest/today.
2. Library functions that exist for those handlers: Generate, FurtherPlanning (planner); Start/Turn/Finish (teach, memory store only); ReviewCard, Override, Due (srs). Library functions that do NOT exist for any endpoint: create/list subjects, get plan for a week, patch plan block, get report by session, edit card, digest. The Planner and TeachEngine interfaces have no method for GET plan, PATCH block or GET report, so INT cannot satisfy the contract without extending domain (forbidden for unit branches; needs Cris).
3. Type fidelity: JSON tags in domain match OpenAPI field names (spot-checked Source, Subject, ClaimResult, ReviewEvent, PlanBlock, Card, DueItem, Report, FurtherPlanning). Differences: OpenAPI `Subject.exam_date` is a date while Go uses *time.Time (will serialize as date-time); OpenAPI `Plan.week_start` is date-time but request uses date; OpenAPI `FurtherPlanning` carries only what the domain type carries (no gaps, no change reasons) although PLAN D7 and U7 prompt require "gaps, next week's hours and the reason for each change" which only `planner.Progress` provides; `Report` has no separate `gaps`/`corrections` fields (results carry them). `DueItem` has no `reason` although U4 prompt says Due returns reasons.
4. Wiring: `cmd/api/main.go` wires nothing; there is no composition root, no pgxpool (srs/planner use small DB interfaces because `puddle` is not in go.mod; INT must add pgxpool, which means touching go.mod, which U0 intended to avoid), no env loading beyond PORT. Fakes are only used inside tests and evalcheck's `--model fake`. No code path currently connects Checker -> Engine -> Scheduler -> Planner with real implementations except in unit tests of individual pieces; the planner store test seeds claim_labels/events by direct SQL.
5. Interface satisfaction is enforced at compile time by `var _ domain.X = (*T)(nil)` in checker, teach, srs, planner (and fakes). domain.Digester has only a fake.

---------------------------------------------------------------------------

## 8. Code quality findings (bugs, rule checks, security, test gaps)

Severity: H = breaks a hard rule or the demo; M = correctness or risk; L = minor.

F1 (H, bug, reproduced). Citation dropped after the checker verified it. checker.verifyCitation compares whitespace-normalized strings (internal/checker/verify.go:35-44), but teach.finalize requires raw `strings.Contains(src, r.Citation.Quote)` (internal/teach/engine.go:299). unit.json source text keeps newlines between lines (content/unit_test.go normalize keeps "\n"), so any quote that spans a line break and is written with spaces fails at Finish. Reproduced in a scratch worktree: source "alpha beta\ngamma delta", quote "beta gamma" -> after finalize `citation=nil correction=nil` while label stays `contradicted`. Effect: the report shows a "contradicted" label with no correction and no quote, and the grade is still Again; the demo's "cited gap" can vanish. Also finalize does not downgrade contradicted to partial when it drops the citation (the checker does), so the two layers disagree on the rule.

F2 (H, gate integrity). The "AI student states 0 untaught facts" bar cannot be demonstrated by the existing suite. leak_test.go uses a fake LLM that returns a canned reply, so it only checks prompt wiring. leak_real_test.go (build tag realllm) uses a toy calculus source, not the MATH unit, uses `fakes.Checker` (no probe hints), has a hand-picked banned-word list, and has never been run. Additionally the probe_hint is written by the checker model, which sees the source, and goes into the student's SYSTEM prompt (internal/teach/student.go:41-42, engine.go:133); the only guard is a 5-word verbatim shingle (student.go:97, verify.go:13), so a hint like "ask about the memoryless property" passes and teaches the student a source term it was never told. By design per the PROMPT, but it is a real leak channel for rule "AI student never receives source material".

F3 (H, completeness). No Postgres teach.Store (see section 3). Without it teach_sessions, teach_turns, taught_claims and claim_labels are never written, so planner.loadExplanation (progress.go) never has data and the "recall and explanation separately" screen and gaps list are empty in a real run. Sessions are in a process-local map (engine.go:33) with no eviction, so a restart mid-demo loses the session ("unknown session").

F4 (H, completeness). No loader for content/unit.json and no content/seed.sql. Nothing inserts sources, concepts or claims into Postgres; scripts/seed.sql only inserts the user and subject. srs.ApplyTeachBack runs `loadClaims` against the DB (scheduler.go) and teach needs LoadConcept, so the loop cannot run on a freshly seeded DB. PROMPT U1 step 7 (history so the demo concept is due ~7 days out and can visibly move earlier) is unimplemented.

F5 (M). Planner blocks are scheduled 09:00 to 13:00 UTC (internal/planner/allocate.go:28,38,281) which is 04:00 to 08:00 Toronto time; the Plan tab would show nonsensical study times. Day windows are UTC. U5 plan lists this as a risk.

F6 (M). Checker outage degrades silently into a punishing grade. If every check fails or times out (engine.go:208-229 logs and returns), Finish uses empty results and `finalize` marks every required claim `absent` (engine.go:311-316), which maps to Again for the concept and creates a gap card per claim, even if the student explained everything correctly. No "checker unavailable" state, no retry at Finish, no fallback to "suggestions only" as PLAN says for a failed gate. Similarly the last turn's check may still be pending after FinishWait=8s (engine.go:249-257) and Finish then grades on an older, shorter transcript. Also Finish does not trigger one final check with AllowAbsent semantics; absent is derived by code from "never seen", which also marks as absent any required claim the model matched under an unknown/nil claim id.

F7 (M). Finish applies `ApplyTeachBack` without an event id (engine.go:267), so DB-level idempotency (WithEventID) is unused; idempotence relies on the in-memory `s.report` cache. A retry after restart double-writes.

F8 (M). Citation verification is weaker than "exact substring of an approved source" suggests: no minimum quote length (one word passes; verify.go:43-44), locator unchecked, and the engine path (finalize) does not re-verify source approval (MemStore.LoadConcept returns all sources; a future pg Store must filter to course + reviewed allowlist, nothing enforces it in teach). Quotes of garbled math (damaged source.md) will be brittle.

F9 (M). Evalcheck does not test the production request: separate adapters, no JSON suffix on Claude, no retry, Gemini path has ResponseMIMEType JSON while production Gemini has no JSON method at all (internal/llm/gemini.go: `JSON` returns ErrUnsupported). If the bake-off picks a Gemini checker, it cannot be deployed without new code. Anthropic.JSON caps output at 2048 tokens (internal/llm/anthropic.go:32 `MaxTokens: 2048`), evalcheck uses 4096 [unverified whether either truncates].

F10 (M). `Gemini` default MaxTokens=120 (gemini.go:32) with no thinking configuration [unverified effect on thinking models]. Sentence cap treats "." followed by whitespace as sentence end (student.go:134 Feed): abbreviations like "e.g. " or "i.e. " end a sentence early (decimals are handled and tested).

F11 (M). Scheduler semantics worth a human decision: (a) Again for a first teach-back gives due +1 day, Good +3 days because short-term steps are disabled (srs/logic.go newFSRS); (b) review of a card grades the whole concept; (c) a concept never reviewed never appears in Due; (d) unmatched ("claim_id null") partial statements affect the grade; (e) FSRS "Hard" on a first look is not "weaker than Again" in schedule terms (Again 1d, Hard 2d), so the visible difference between a wrong and partial answer is one day. For the demo the wrong-answer effect on a concept already scheduled 7 days out is large; on a new concept it is small.

F12 (M). `cmd/migrate` has no advisory lock (comment admits it) and applies each file in one tx; fine for single dev DB. Test suites leave rows in the shared dev DB and cannot clean them because of the append-only trigger (planner tests use rolled-back transactions to avoid this; srs tests do not).

F13 (M, process). U1 to U5 landed on qa without PRs (history shape, section 1.1), branches are not protected, and the guard hook is the only enforcement. The "Jev split" (regulation 3) was never run; PLAN open items (demo unit rights, allowlist pack, owner confirmation) still unchecked.

F14 (L). evalcheck.loadLabels index panic on short rows (cmd/evalcheck/eval.go:87). cmd/api has no timeouts on http.Server (ListenAndServe default), no CORS. extractJSON only strips fences (internal/llm/anthropic.go) although its comment says "first JSON value"; trailing prose after the JSON object makes decode fail (and decode failures are not retried). `fakes.Scheduler.ReviewCard` hardcodes "concept-1".

F15 (L). Hard-rule checks passed in code: review_events has no UPDATE/DELETE code and a DB trigger (verified live); student reply capped and question-only by prompt (real behavior [unverified]); transcript/latex and taught text are fenced as untrusted data for both models; `StudentContext` field list locked by test; all SQL parameterized; no secrets committed; no em dashes; no learning-gain claims found in code or product strings (PLAN.md states none).

Test gaps: no test of the HTTP layer (none exists); no real-model test run anywhere; no concurrency test for Turn/Finish overlap on a Postgres-backed store; no test that ties the real Checker output through teach.finalize with multi-line sources (the F1 gap); no test of planner `Generate` across a DST change or non-UTC; no test that gap cards are editable (no edit path); no digest tests (no digest); cmd/migrate untested; cmd/api and evalcheck coverage 50% and 42%; load test and Playwright not present.

---------------------------------------------------------------------------

## 9. What is blocking the closed loop (ordered)

1. Cris approval of the concept list/demo unit; U1 remaining outputs (eval/data/explanations.jsonl with 20 explanations, labels files, check_labels.py, labels_final.csv by Cris and Dino, content/seed.sql with history).
2. API keys and model ids (STUDENT_MODEL, CHECKER_MODEL not in .env.example) and a first real run of: checker via evalcheck (Gemini tier and Claude tier), student leak suite.
3. Fix F1 and decide F6 behavior before the gate.
4. INT work: pgxpool + composition root, teach pg Store (incl. claim_labels writes), unit loader/seed, HTTP handlers for all 14 endpoints including SSE, domain extension for the missing methods (needs Cris), digest (U6) and UI (U7).
5. GATE, QA, RELEASE per rpi prompts; fix docs/TESTING.md B4 and gate.md naming first so Dino's checklist is passable.

## 10. Memory-derived context

None of the facts above came from the user's memory store; the repository, git history, GitHub API and local test runs were the only sources. The docs name Dino (GitHub dodino24) as the labelling and testing partner and Cris as implementer.
