# Space Repetition Hackathon Plan

Nov 9, 2026 · @Cris

**Build one closed loop on one course unit in under 8 hours: voice teach-back finds gaps, gaps change the review schedule, and the schedule changes the week's plan.** Do not merge the three repos. Status: waiting for approval from Cris or Dino under regulation 5.

## Honest evaluation

**Verdict: accept with revisions, worth pursuing pending the Hour 3 checker test.** The feature bundle is not new. The defensible idea is teach-back results changing what gets reviewed and when.

| Original idea | What the research found | Decision |
| --- | --- | --- |
| Replicate ai-study-planner | The visible plan is `ceil(topics/days)` over hardcoded lists. The LLM output is saved but never shown. | Rebuild. Use its dashboard layout as a reference only. |
| Replicate Obsidian Spaced Repetition | Solid logic, but schedules live inside note files and FSRS covers flashcards only. | Rebuild storage around a review log. Keep card syntax and sibling burying ideas. |
| Use OpenMAIC as the AI student | The human is the learner. Its students are told "You are NOT a teacher." No per-agent knowledge state exists. | Do not fork. Borrow its `feynman-learning` protocol. |
| One AI that plays student and corrects | LLMs told to act low-mastery still scored 96.8 to 100%, and they improve after wrong teaching too. | Split into a restricted student and a separate grounded checker. |
| Self-rated level drives the plan | Self-rating is not a measurement. | Level seeds week one, then measured concept state takes over. |
| One notification per subject | Reminder evidence is mixed, and some studies are negative. | One daily digest listing each subject. |

**Fatal-flaw audit.** No flaw is critical. Two are major.

| Flaw | Severity | Defense |
| --- | --- | --- |
| The bundle already exists: AeraWise lists all four parts, and Synapsi and RemNote cover most of them | Major | Claim only the closed loop. Never pitch "first to combine these." |
| Learning benefit is unproven: the best controlled LLM-tutee study (N = 96) found no post-test difference, p = .21, and no study read used a delayed test | Major | Do not claim learning gains today. Measure with a delayed, unaided test after the hackathon. |

**Scores, starting at 5 and adapted from research to product.** All are mechanism-based and not confirmed by data.

| Dimension | Score | Evidence | How to raise it |
| --- | --- | --- | --- |
| Higher (learning) | 6 | Teaching with expectancy g = 0.48, but LLM evidence is mixed | Delayed-test pilot |
| Faster (study time) | 6 | Time moves to the weakest concepts by design | Compare minutes per retained concept |
| Stronger (reliability) | 7 | Knowledge state lives in code, and every correction needs a citation | Pass the Hour 3 gate |
| Cheaper | 7 | Modeled at $0.004 to $0.07 per 15-minute text session, before the separate checker roughly doubles it | Run the model bake-off |
| Broader | 6 | Works for courses and allowlisted outside topics such as coding interviews | Test a second domain |

**Biggest competitive threat:** Gemini Study notebooks already have a plan, tracker and flashcards. **Biggest product risk:** students may avoid teaching because it is effortful. Only about 15% of students with Khanmigo access engage regularly.

**Limit of today's demo:** it can prove the loop works. It cannot prove students learn more.

## Goals and non-goals

**Goal:** a live demo where a wrong explanation visibly changes tomorrow's review and next week's plan.

**In scope**

- Add subjects with weekly hours, level and exam date, then generate an editable weekly plan.
- Show each teach-back on the plan ahead of time, so the student studies expecting to teach.
- Teach by voice, with typed LaTeX when formulas need it.
- AI student asks one question per turn and knows only what it was taught.
- Checker labels each claim and cites the course unit or an allowlisted source.
- Gaps become cards and FSRS updates each concept's next review. The student can move any date.
- Further Planning tab shows recall and explanation separately for each subject and concept, with next week's hours.
- One daily digest by subject, skipped once the student has studied.

**Out of scope today**

- Live PDF upload. The unit is parsed once before the demo.
- Accounts, billing, mobile apps and offline sync.
- Users under 18.
- Per-user FSRS optimization.
- Any claim about learning gains.

**If time runs short, cut in this order:** digest delivery, outside-source pack, plan editing, generated cards. Never cut the checker or the schedule update, because they are the product.

## Confirmed decisions

All of these came from Cris's answers on 2026-11-09.

| Area | Decision | Reason |
| --- | --- | --- |
| First users | Cris, Dino and classmates | Real use tests the idea fastest |
| Repos | Rebuild features, borrow ideas | Their code does not fit the design or the backend rule |
| Review unit | Concept, holding cards, problems and teach-back | Fits math and economics, and gives per-concept performance |
| Knowledge sources | Course material plus a curated allowlist | Supports course study and self-study such as coding interviews |
| Plan inputs | Hours, level and deadlines, then measured data | Useful on day one, more accurate over time |
| Scheduling | FSRS with manual override | Adapts to performance, and the student keeps control |
| Teaching input | Voice first, typed LaTeX when needed | Natural Feynman practice, with a route for formulas |
| Notifications | One daily digest by subject | Avoids alert fatigue |
| Backend | Go, with Python only for PDF parsing | LLM wait dominates request time, and Go handles many streams cheaply |
| Demo scope | Closed loop on one course, material pre-loaded | Removes the riskiest live failure |
| Rules today | Written UI exception, Git flow unchanged | A demo needs a UI, and the review trail stays |
| AI access | Google Gemini API, Anthropic API, browser speech | Available today |

## The closed loop

&#91;embedded content: closed loop · 7 parts\]

The checker is the highlighted part because every later step trusts its labels. It is the first thing tested.

## System design

**A single Go service owns all state and rules. LLMs only write words and label claims.** This keeps the AI student honest and keeps LLM latency out of every other request.

| Layer | Choice | Reason | Trade-off |
| --- | --- | --- | --- |
| API | Go `net/http` with server-sent events | One-way streaming is enough, and it needs little code | Replies stream one way only |
| Data | Postgres with an append-only `review_events` table | Analytics and later FSRS fitting need full history | Needs a local Docker service |
| Scheduler | go-fsrs (MIT), FSRS-6 default parameters | Same language as the backend, and defaults already beat classic SM-2 for 98.1% of benchmark users on prediction | Optimizer unfinished, which can wait |
| Grounding | Whole unit sent as cached context today, pgvector later | One unit is small, so retrieval adds failure with no gain | Does not scale past one unit |
| PDF parsing | One-off Python script, later a queue worker | Never touches the request path | A second language, limited by exception |
| AI student | Cheap Gemini tier | High turn volume with low reasoning needs | Exact model picked by bake-off |
| Checker | Claude Sonnet 5.5, unless the bake-off says otherwise | Accuracy matters most here | Costs more per call. This plan's author is a Claude model, so the bake-off decides. |
| Speech | Browser speech input and output, Chrome only | Free and available now | Weak on spoken math, so a transcript edit box and LaTeX input are required |
| Demo UI | React, Vite, KaTeX, thin layer on the API | Written exception to "UI deferred" | Will be redesigned in the real UI phase |

**How the AI student stays ignorant.** The student model receives only claims the human already taught, plus one probing target chosen by code. It never receives the source. The checker sees the source and the transcript, and it runs after each turn without blocking the student's reply.

**Correction timing.** The AI student questions during the session, and it does not lecture. Cited corrections appear in the report after the attempt, because retrieval with feedback works best and a student who corrects its teacher breaks the role.

**Grade mapping, per concept.** This is an experiment and is logged as its own event type.

| Worst result on a required claim | FSRS grade |
| --- | --- |
| Contradicted or absent | Again |
| Partial | Hard |
| All supported | Good |

Teach-back never grades Easy, since one good explanation is weak evidence of long retention.

**Plan allocator.** Each 30-minute block goes to the highest-scoring subject, then scores are recomputed. Level multiplies the score in week one and fades as data arrives. A student-set share of hours, proposed default 30%, is reserved for new material. No published allocator of this type was found, so this is a design proposal.

```latex
\text{score}_s = w_s \cdot \text{level}_s \cdot \text{urgency}_s \cdot \max(0,\ R_{\text{target}} - \bar{R}_s(t_{\text{exam}}))
```

**Core tables:** `subjects`, `concepts`, `sources` (trust tier, reviewer, review date), `teach_sessions`, `claim_labels`, `cards`, `review_events` (card review, teach-back or manual override), `plan_blocks`, `digests`.

**Performance targets.** Proposed numbers need human approval under regulation 1, and all are unmeasured until QA.

| Path | Target |
| --- | --- |
| Non-LLM reads, p95, 50 concurrent users | Under 50 ms |
| Schedule and plan recompute for one user | Under 20 ms |
| AI student first token after the human stops | Under 1.5 s |
| Checker result after a turn, asynchronous | Under 6 s |

QA tests non-LLM paths with a mock LLM, so results measure our code and not vendor latency.

## Work split

Eight independent units run from one API contract, so no two touch the same files. Hours count from plan approval and total 7.5, leaving 30 minutes spare. Cris owns implementation and coding and runs the subagent sessions. Other human owners are proposals until confirmed.

| Unit | Owner | Files | Start to end (hour) | Effort |
| --- | --- | --- | --- | --- |
| U0 Setup: repo, branches, Docker, API contract, migrations | Cris with main agent | `api/`, `migrations/`, `compose.yaml` | 0 to 0.5 | Medium |
| U1 Concept pack and allowlist: parse unit, list concepts and claims, label eval set | Cris and Dino, parser subagent | `content/`, `scripts/parse_pdf.py`, `eval/data/` | 0.5 to 2 | High, human judgment |
| U2 Teach engine: knowledge state, student prompt, streaming | Subagent | `internal/teach/`, `internal/llm/` | 0.5 to 3 | High |
| U3 Checker and bake-off | Subagent | `internal/checker/`, `eval/` | 0.5 to 3 | High |
| U4 Scheduler: FSRS wrapper, grade mapping, overrides, cards | Subagent | `internal/srs/` | 0.5 to 2.5 | Medium |
| U5 Planner: allocator, analytics queries | Subagent | `internal/planner/` | 0.5 to 2.5 | Medium |
| U6 Daily digest: in-app list and browser notification | Subagent | `internal/digest/` | 3.5 to 4.5 | Low |
| U7 Demo UI: Plan, Teach, Review, Further Planning tabs | Subagent, Dino reviews | `web/` | 1.5 to 5 | Medium |
| Integration | Main agent | `cmd/`, wiring only | 3 to 5 | Medium |
| QA on `qa` branch: tests, load test, full demo run | Subagent and Cris | `qa/` | 5 to 6.5 | Medium |
| Release to `main`, rehearsal, submission | Cris and Dino | None | 6.5 to 7.5 | Low |

**Effort tags are manual.** Regulation 3 requires Jev for this split. No Jev or OpenRouter key was found in this environment, so see the open questions.

**Git.** `main` is production and `qa` is the test branch. Work happens on `feature/<short-name>`, one concern per PR, merged to `qa` first. Only QA-approved changes move to `main`.

**RPI workflow.** Each unit has `rpi/<unit>/research.md`, `plan.md` and `impl-log.md`. A subagent can start from those files alone and reports back in under 200 words.

## Acceptance criteria and risks

**Hour 3 gate decides whether the core works.** The eval set is 20 prepared explanations, labelled by Cris and Dino: 10 correct, 5 partial, 5 wrong. Thresholds are proposals for human approval.

| Check | Proposed pass bar |
| --- | --- |
| Wrong explanations flagged contradicted or absent | At least 4 of 5 |
| Correct explanations falsely flagged contradicted | At most 1 of 10 |
| Corrections with a valid citation | 100% |
| AI student states facts it was never taught, 10 probes | 0 |

**If the gate fails:** narrow the concept list and switch the checker model once. If it still fails, show gaps as suggestions, stop them changing the schedule, and say so in the demo.

**Demo acceptance**

- [ ] A student adds at least two subjects and gets a weekly plan.
- [ ] A voice explanation with one planted mistake produces a cited gap.
- [ ] That concept's next review date moves earlier, and the change is visible.
- [ ] The gap creates at least one editable card.
- [ ] Further Planning shows recall and explanation separately, and next week's hours change.
- [ ] The daily digest lists each subject with due count and minutes.
- [ ] A manual date override is saved and logged.
- [ ] A full demo run passes on `qa` before release to `main`.

| Risk | Level | Mitigation |
| --- | --- | --- |
| Checker misses subtle math errors | High | Hour 3 gate, small concept list |
| Spoken math transcribes badly | High | Editable transcript, LaTeX box, explain concepts in words on stage |
| Eight hours is too short | High | Cut order above, parallel units, 30-minute spare |
| Venue wifi or API failure | Medium | Recorded fallback run and cached responses |
| Mapping checker labels to FSRS distorts scheduling | Medium | Logged separately as an experiment |
| Slides are copyrighted, or data goes to US APIs | Medium | Use material you have rights to, and disclose processing. Legal position not researched. |
| Chrome speech sends audio off device | Low today | Disclose it, and test behavior at Hour 0. Unverified. |

## Required planning artifacts

Five of the six regulation 6 artifacts are drafted here or in the research report. Design research is missing, and all need agreement before the build starts.

| Artifact | Status | Where |
| --- | --- | --- |
| Git branch strategy | Drafted | Work split |
| System design with performance focus | Drafted, targets unmeasured | System design |
| Work split for subagents | Drafted, Jev not yet applied | Work split |
| Market research | Done, with unverified items marked | AI study product evaluation report |
| Design research | Not done. Proposed: Dino writes a 30-minute note on 3 dashboards. | Open |
| UI strategy | Drafted as a thin demo UI, under exception | System design |

**Exception to log under regulation 10, if approved:** on 2026-11-09, a thin demo UI may be built alongside the backend for the hackathon. Git flow is unchanged.

## Open questions

These need an answer from Cris or Dino before implementation starts.

- [x] Approve this plan, the proposed gate thresholds and the performance targets?
- [x] Confirm the correction change: the AI student only asks questions, and cited corrections come after the attempt?
- [ ] Which course unit is the demo unit, and do you have the material and the right to use it? MATH 6602 Chapter 2, Markov chains, appears in the Study - F26 sheet. Use MATH from Dino, as a test data
- [ ] Which single outside-source pack goes on the allowlist first, for example one coding-interview topic? Leetcode, Neetcode, Algo monster, AlgoExpert,...
- [ ] Who is on the team today, and are the proposed owners right? Cris and Dino, Dino done the plan, Cris will handle the code
- [ ] Exact submission time, required deliverables, judging criteria, and any rule on prewritten code? Afternoon 5pm
- [ ] Provide a Jev key as an environment variable, or approve a manual split for today and log it? Cris's computer already has Jev, please have a look
- [ ] Where does the code live: new GitHub repo name and owner? Cris will create repo and we can start edit it
- [ ] Repo: <https://github.com/CrisH2307/repetition-study>

## Sources

Repo claims come from code audits at pinned commits. Nothing was run. The OpenMAIC prompt quotes were rechecked against the audit notes. A direct recheck of the go-fsrs and OpenMAIC pages was blocked in this session, so those rest on the audits. Effort estimates are unvalidated.

- [ai-study-planner](https://github.com/Harshal-Bsys27/ai-study-planner), [obsidian-spaced-repetition](https://github.com/st3v3nmw/obsidian-spaced-repetition), [OpenMAIC](https://github.com/THU-MAIC/OpenMAIC)
- [go-fsrs](https://github.com/open-spaced-repetition/go-fsrs), [srs-benchmark](https://github.com/open-spaced-repetition/srs-benchmark/tree/main/result)
- [AeraWise](https://aerawise.com/), [Synapsi](https://synapsi.guru/), [RemNote Exam Scheduler](https://cf.remnote.com/feature/exam-scheduler), [Gemini for Students](https://gemini.google/us/students/?hl=en-US)
- [Khanmigo engagement](https://www.edtechinnovationhub.com/news/only-15-percent-of-students-with-access-to-khanmigo-actually-use-it-khan-academy-admits)
- [Xu et al., LLM tutee comparison](https://arxiv.org/abs/2601.16583), [SSKG, simulated mastery](https://arxiv.org/abs/2608.21668), [Do et al., feedback flips](https://arxiv.org/abs/2605.12748)
- [Kobayashi 2024, teaching expectancy](https://consensus.app/papers/details/cbd8be5276e9576fa9c207c49c0ea94b/?utm_source=claude_desktop)
- Full citations are in the AI study product evaluation report.
