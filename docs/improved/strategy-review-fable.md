# Space Repetition: strategy and architecture review

Date: 2026-11-09. Author: Claude (Fable 5.2), acting as lead strategist and architect. Audience: Cris and Dino.

Status of this document: a proposal. Under regulations 1 and 5, nothing here is decided until Cris or Dino confirms it. Every numeric bar is a proposal. Items I could not verify are tagged [unverified]. I am a Claude model, so any recommendation that favours a Claude model for a role must be settled by the bake-off, not by me.

Inputs read in full: `/home/claude/gather/code_dossier.md`, `planning_dossier.md`, `market_and_repo_research.md`, `project_regulations.md`, `plan_doc_tab1.txt`; in the repo (`origin/qa` at 8488715): `CLAUDE.md`, `docs/PLAN.md`, `docs/TESTING.md`, `.claude/boundaries.json`, `rpi/*/PROMPT.md`, plus spot checks of `internal/teach/engine.go`, `internal/teach/student.go`, `internal/checker/checker.go`, `internal/checker/verify.go`, `content/source.md`, `content/unit.json`, `content/damaged-formulas.md`, `migrations/0001_core.sql`, `internal/domain/interfaces.go`. Market data was refreshed by web search and page fetches today; sources are listed in section 9.

Known blind spot: no source covers the hours between 13:18 and the 17:00 deadline. If INT, U7 or the gate were worked on after 13:18 and not pushed, the readiness scores below are too low. Cris should correct this first.

---

## 0. Verdict in one page

**Verdict: keep going, but as a validation project with a hard evidence gate, not yet as a company.** The kernel is good engineering. The thesis it exists to test (a checker can reliably catch wrong explanations of graduate math and prove it with a quote) has never been run once against a real model. Until that gate is run, everything else is decoration.

Top five recommendations:

1. **Run the real gate before writing another feature.** Fix the citation bug (F1), build the claim-level eval set with Dino as the domain labeller, make `evalcheck` call the same code path as production, and run Claude and Gemini tiers. Two weeks, then decide.
2. **Narrow the wedge to proof- and derivation-heavy courses** (graduate and upper-year math, statistics, economics), Ontario first. Every funded competitor targets flashcard-shaped content (AP, MCAT, IGCSE). None was found that verifies a correction against a quote from the student's own course notes with math rendered correctly.
3. **Change four design decisions**: typed LaTeX first instead of voice first; one final check at Finish instead of a strong-model check per turn; claim-level memory state instead of concept-level; probe questions from a code-owned taxonomy instead of free-text hints written by the model that saw the source.
4. **Fix the content pipeline, because it is the real product bottleneck.** `pdftotext` output is unusable for math (`content/source.md` page 2). Move to vision-model transcription into block-addressed Markdown plus LaTeX with a human verification step, and cite by block.
5. **Treat course-material rights as a launch blocker now.** A professor's full practice set is committed to the repo (`content/source.md`, `content/unit.json`), although the plan said it would live in a gitignored `content/private/`. York's copyright office states instructors own their teaching materials and that uploading them to commercial third-party sites "would qualify as copyright infringement". Store uploads privately per user, never share source text between users, and get the professor's permission for the pilot.

Top decisions needed from Cris and Dino (full list in section 6): the goal (validated tool versus business), the target segment, typed-first versus voice-first, claim-level scheduling, what happens to the committed course material, lifting the "UI deferred" regulation for the MVP, and the revised gate bars.

---

## 1. Honest evaluation

### 1.1 Where things stand

| Area | State on `origin/qa` | Evidence |
| --- | --- | --- |
| Domain contract, schema, fakes | Done, frozen, with 9 known contract gaps | `internal/domain/`, `migrations/0001_core.sql`; code dossier section 3 |
| Scheduler (FSRS wrapper) | Done as a library, Postgres tests pass, 3.4 ms per 1,000 concepts | `internal/srs/`; code dossier section 6 |
| Planner | Done as a library, wrong time zone, unconfirmed hours semantics | `internal/planner/allocate.go:28,154,281` |
| Checker | Done as a library with fakes; never called a real model | `internal/checker/`; `rpi/U3/impl-log.md` |
| Teach engine | Done against an in-memory store only; no Postgres store | `internal/teach/store.go`, `internal/teach/engine.go:33` |
| LLM clients | Tested against stub HTTP servers only | `internal/llm/llm_test.go` |
| Content pack | Draft, unapproved, math damaged, claims are worked-example numbers | `content/unit.json`, `content/damaged-formulas.md` |
| Eval data and labels | Do not exist | no `eval/` directory |
| API | 1 of 15 paths served (`/healthz`) | `cmd/api/main.go:13-19` |
| UI, digest, integration, QA, release | Not started | empty `rpi/U6`, `U7`, `INT`, `QA`, `RELEASE` files |
| Hour 3 gate | Never run | `rpi/GATE/` empty |

### 1.2 Readiness scores (0 to 10, my judgment)

| Dimension | Score | Why |
| --- | --- | --- |
| Engineering kernel (libraries, tests, rules in code) | 7 | 61 tests pass, race-clean, append-only log enforced by trigger, hard rules encoded |
| Closed loop working end to end | 1 | No composition root, no store, no loader, no handlers |
| Core thesis proven (checker works on real math) | 0 | Zero real model calls, zero labels |
| Content pipeline | 2 | Parser output unusable for formulas, claims hand-drafted and unapproved |
| Product surface (UI, auth, deploy) | 0 | None exists |
| Process discipline | 5 | Strong kit and file boundaries held; PR rule broken for U1 to U5; no branch protection |
| Market validation | 1 | One real user, whose own tracker has no entry since 29 Oct |
| **Overall readiness to compete** | **2 of 10** | A strong prototype kernel with an untested thesis |

### 1.3 What is strong

- **The architecture principle is right and rare.** "Code owns state, LLMs only write words and label claims" (`CLAUDE.md`) is the correct answer to the two research failure modes (simulated ignorance does not work; models flip to correct answers regardless of teaching quality). `StudentContext` has no source fields and a reflection test locks that (`internal/teach/student.go:15-21`).
- **Citations are verified by code, not trusted.** `internal/checker/verify.go:35-44` rejects any correction whose quote is not in an approved source. This is the single most defensible technical property and the base of the trust story.
- **The review log is append-only at the database level** (`migrations/0002_review_events_append_only.sql`), which is what later FSRS fitting, analytics and any study will need.
- **The team wrote down what it does not know.** `docs/PLAN.md` says the demo cannot prove students learn more. `CLAUDE.md` bans learning-gain claims. That honesty is a market asset when every competitor page claims mastery.
- **File boundaries held.** Every unit branch touched only its allowed paths (code dossier section 1).

### 1.4 What is wrong

1. **The gate that the whole plan hangs on was skipped, and the build continued anyway.** `docs/PLAN.md` says "worth pursuing pending the Hour 3 checker test". The test needs `eval/data/`, labels and keys; none exist. Units U4 and U5 were still built on top of an unproven checker.
2. **Bug F1 breaks the headline feature.** The checker verifies a quote after whitespace normalization (`internal/checker/verify.go:32-44`), then `finalize` re-checks with raw `strings.Contains` (`internal/teach/engine.go:299`). Source text has line breaks, so a verified citation is dropped at Finish and the report shows "contradicted" with no quote. I confirmed the code path by reading both files.
3. **The "AI student never receives source material" rule has a hole (F2).** The checker, which sees the source, writes a free-text `probe_hint` that goes into the student's system prompt (`internal/teach/student.go:39-42`). The only guard is a 5-word shingle check. A hint such as "ask about the memoryless property" passes and hands the student a source term.
4. **A checker outage punishes the learner (F6).** If checks fail or time out, `finalize` marks every required claim `absent` (`internal/teach/engine.go:311-316`), which grades the concept Again and creates a gap card per claim. A vendor incident would silently corrupt every schedule touched that day.
5. **The eval harness does not test production (F9).** `cmd/evalcheck/llm.go` has its own adapters and a different prompt suffix from `internal/llm/anthropic.go`. A pass in evalcheck would not certify the shipped checker.
6. **The demo content does not match the plan.** The plan's planted mistake is about discrete-time chains ("irreducible means every state has period 1"); the unit is a continuous-time practice set (`rpi/U1/research.md`). The claims are mostly worked answers such as "P(N(1) >= 10) = 0.0839", which turns teach-back into reciting numbers.
7. **The original gate bars cannot support a decision.** "At least 4 of 5" and "at most 1 of 10" are too small to mean anything: 4 of 5 is consistent with a true detection rate anywhere from about 28% to 99% (95% interval). They were reasonable for a hackathon and are not reasonable for a product.
8. **The labelling plan uses the wrong second labeller.** Cris and Dino were to label independently. The material is graduate stochastic processes. Cris is not a domain expert in it [my assumption, confirm], so disagreements would measure knowledge, not label quality.
9. **Planner places study blocks at 04:00 to 08:00 Toronto time** (`internal/planner/allocate.go:28,281`, UTC day start).
10. **Decisions live in three places that disagree.** The plan doc, the Claude Project docs and the repo each hold a version. Dino's chat at 12:37 was answered from stale project docs and told him no code existed when five units were already merged (planning dossier section 4.4). This will keep happening without one source of truth.
11. **Regulation 4 was broken on day one.** U1 to U5 reached `qa` by local merges without PRs, and U1 landed while its own commit message said it awaited approval. No branch is protected on GitHub.

### 1.5 Assumptions I am challenging

| Assumption (source) | Challenge | What I recommend |
| --- | --- | --- |
| Voice first (`docs/PLAN.md`, confirmed decision) | The first real user studies graduate math and economics. Browser speech recognition is Chrome-only and weak on formulas; the plan itself rates this risk High. Dino's own notes are formulas (planning dossier section 2). | Typed Markdown plus LaTeX with live KaTeX first; microphone as an optional input into the same editable box |
| The checker should run after every turn with a strong model (`internal/teach/engine.go`) | It multiplies cost by roughly 4 (section 4.5), creates the Finish race in F6, and is only needed to pick the next question | One full check at Finish; a cheap per-turn "which claims were touched" matcher that never sees source prose |
| The concept is the FSRS unit; worst claim sets the grade (`docs/PLAN.md` grade mapping) | FSRS models atomic items. A concept with five claims where one is missing is graded Again, the same as knowing nothing. Card reviews also regrade the whole concept (`rpi/U4/research.md`) | Keep memory state per claim; roll up to the concept for display and planning |
| A weekly calendar of 30-minute blocks is part of the wedge (earlier research ranked it second) | Dino's tracker has no time-of-day planning at all. He tracks a next date per chapter and an error log. Timed blocks are the least validated and buggiest part | Ship a "Today" queue across courses with estimated minutes and a per-course weekly budget. Keep the allocator library, hide timed blocks |
| The review unit is concept plus cards (confirmed decision) | Dino's sheet is an error log at the problem level ("A1: 9 - formula of eigenvalues, eigenvectors, adjugate of a matrix"). The product has no place for "this assignment question went wrong" | Add a problem or error item type that attaches to a concept and can open a teach-back |
| Cris labels half the eval set | See 1.4 item 8 | Dino plus one classmate label; Cris builds tooling |
| "Exact substring" is enough for trust (`CLAUDE.md`) | A one-word quote verifies (F8), and a quote of garbled math verifies too. Exactness of a meaningless string is not trust | Cite by block id, require a minimum quote length or a whole formula block, show the rendered block and page image |
| Outside-source allowlist with LeetCode, NeetCode and others (Cris's edit in the plan doc) | Those are proprietary sites; storing their text to verify quotes is a licence problem, and it dilutes the wedge | Drop from v1. A second domain later should use material the user owns |
| Canadian students may not get Google's free offer (earlier research, tagged unverified) | Now verified: Google announced on 28 Oct 2026 one free year of Google AI Plus for eligible Canadian post-secondary students, claimable until 31 Dec 2026, including study notebooks, flashcards and practice quizzes | Assume every pilot user has Gemini study tools for free. Compete on what Google does not do |
| "No product has been confirmed to ship the loop" (earlier research) | Still not disproven, but the gap is closing. Synapsi now ships an "AI-judged Feynman teach-back" that "judges your answer against the source and shows your gaps" with FSRS, at $9.99 a month, and shipped version 1.37.1 within the last week. | Stop positioning on the loop alone. Position on rigor for math-heavy courses and on verified quotes |
| The AI student's knowledge model is the product (earlier research conclusion) | In this design the student only asks questions, so its knowledge state barely matters to the learner. What the learner feels is the checker's accuracy and the quality of the report | The checker and the content pipeline are the product. The student is the interface |

---

## 2. Market update (as of 2026-11-09)

Method: web search plus page fetches today. Nothing was tested hands-on. Vendor pages describe intent, not quality. Items I could not open are marked.

### 2.1 Competitors

| Product | What it does that overlaps | Price | What I could not find it doing | Source |
| --- | --- | --- | --- | --- |
| **Synapsi** (solo developer, Khoa Tran) | "AI-judged Feynman teach-back that grades whether you can explain the idea in your own words"; judged "against the source"; reviews "now scheduled by FSRS" (the listing also still says SM-2) | Pro $9.99 a month, $48 a year; lifetime option; lecture credits | Verified quotes, math rendering, cross-course planning. Too few ratings to display. Site `synapsi.guru` was not fetchable from this session | https://apps.apple.com/app/synapsi-ai-study-notes/id6780159251 |
| **AeraWise** | "Teach Lab": explain topics to "Lab, an AI student", "AI-graded Feynman reports", Lab "asks doubts"; exam planner, weakness map, Gap Map, reminders, spaced repetition (algorithm not stated) | "Free", no tiers shown | Any statement that teach-back results change the schedule; math support; company identity; user counts. Targets CBSE, ICSE, JEE, NEET, IGCSE, A-Level, IB | https://aerawise.com/ |
| **Gemini study notebooks and Gemini Notebook (formerly NotebookLM)** | Diagnostic quiz, "tailored learning plan", "skill-based progress dashboard" with Strengths and Focus areas, quizzes "based on class materials you uploaded"; Sept 2026 added voice conversation with notebooks, lecture recorder, short-answer quizzes, asking chat about quiz performance | Free tier; Canada: one free year of AI Plus then $6.99 a month [currency not stated on the page, presumably CAD] | Spaced repetition, teach-back, verified quotes as a grading basis | https://blog.google/innovation-and-ai/products/gemini-app/gemini-study-notebooks/ ; https://blog.google/innovation-and-ai/products/gemini-notebook/new-study-tools-september-2026/ ; https://blog.google/intl/en-ca/products/explore-get-answers/eligible-canadian-post-secondary-students-can-get-one-year-of-gemini-on-us/ |
| **ChatGPT study mode** | Socratic guidance; Aug 2026 "College Student" plugin for Edu builds study guides, quizzes, flashcards; voice over files | Free in consumer plans [from my knowledge of the July 2025 launch; help page not fetchable today] | Persistent schedule, spaced review, teach-back grading | https://learn.chatgpt.com/docs/whats-new |
| **Claude learning mode** | Guides the student to do the thinking instead of answering | In Claude plans [not re-read today; launch coverage only] | Scheduling, persistence of weaknesses, grading against a course source | https://www.engadget.com/ai/anthropic-brings-claudes-learning-mode-to-regular-users-and-devs-170018471/ |
| **RemNote** | Exam Scheduler, FSRS, AI flashcards from PDFs, AI grading | Free; Pro $8 a month; Pro with AI $18 a month (third-party listing) | Teach-back | https://toolradar.com/tools/remnote/pricing |
| **Knowt** | Free Quizlet alternative, AI voice tutor on files, AP and SAT prep | Ultra $24.99 a month on one listing, $9.99 a month billed annually on another [conflicting, unverified] | Teach-back, verified quotes | https://toolradar.com/tools/knowt/pricing ; https://www.techraisal.com/blog/gizmo-ai-vs-knowt-i-used-both-for-a-full-semester-and-who-actually-wins-in-2026/ |
| **Gizmo** | AI flashcards and quizzes with streaks, lives, leaderboards; about 13 million learners; $22M Series A closed 15 Apr 2026; moving into US college | About $77 a year or $6.99 a week (third-party review) | Teach-back, rigor; review says free tier "interrupts your session after wrong answers" | https://doolpa.com/news/gizmo-22m-series-a-gamified-learning-april-2026 |
| **Quizlet** | Deck library; replaced Q-Chat with a ChatGPT-integrated experience in early 2026; Learn mode capped on free | Plus $7.99 a month [from a competitor's blog, unverified] | Teach-back | https://www.mindomax.com/spaced-repetition-apps-updates-2026 |
| **Anki and FSRS** | Default scheduler FSRS-6 since 25.07; AnkiHub took over stewardship in Feb 2026; FSRS-7 described as likely the final version [competitor blog, unverified]; June 2026 team update mentions a new launcher and logo, no AI | Free desktop | Any AI checking; card creation remains manual | https://forums.ankiweb.net/t/updates-from-the-anki-team-june-2026/70120 |
| **"Feynman AI" app cluster** (feynman-ai.com, getfeynman.app and at least six similar App Store listings) | Import PDFs and lectures, generate notes, quizzes, flashcards; some ask you to "explain it simply" and list gaps | Freemium, prices mostly not shown | Source-verified grading, named scheduler, math | https://feynman-ai.com/ ; https://getfeynman.app/ |
| **Alice** | AI tutor, study plans, written and oral mock exams with grading; "more than 250,000 students" | Free tier and Pro (price not shown) | Spaced repetition, teach-back | https://carrot.alice.tech/en-gb/pricing/ |

### 2.2 What this changes from the earlier research

1. **"Teach the AI" is now a named feature in several shipping products.** It is no longer a differentiator by itself. Calling the product "Feynman" anything puts it in a crowded App Store shelf.
2. **The free baseline rose.** A Canadian student can get Gemini study notebooks, quizzes on their own files and a progress dashboard at no cost for a year. A paid product must do something that baseline visibly cannot.
3. **Nobody found today claims verified citations for grading, or math-grade rigor.** Competitor pages are aimed at exam boards, AP, MCAT and general note summarising. No page mentioned LaTeX, proofs or theorem conditions. This is absence of evidence from public pages, not proof of absence.
4. **Synapsi is the closest competitor and it is one person shipping weekly.** Feature parity on the loop is a matter of weeks for them. Rigor on hard material and published evaluation are slower to copy.

### 2.3 What users complain about

Evidence here is thin and mostly second-hand; treat as directional.

- **Trust:** students will not rely on a card they "can't trace back to a source"; ChatGPT can "confidently produce wrong answers and wrong flashcards" (vendor blog summarising Reddit, not verbatim posts: https://scholarly.so/blog/best-ai-for-studying-reddit-2026).
- **Paywalls that appear after adoption:** tools that "look free, then paywall the one feature you needed"; Quizlet Learn and Test moved behind a subscription (same source). Gizmo's lives system is called "the worst paywall design in the category" by one reviewer (https://www.techraisal.com/blog/gizmo-ai-vs-knowt-i-used-both-for-a-full-semester-and-who-actually-wins-in-2026/).
- **Accuracy needing manual review:** reviewer estimates of about 85% (Gizmo) and 90% (Knowt) card accuracy, origin of those numbers unclear [unverified] (same review).
- **Effort:** Anki's learning curve and slow card creation (Scholarly blog). Our product asks for more effort than any of these, which is the main adoption risk.
- **Passive tools get less praise** than tools that force retrieval (Scholarly blog).

No direct Reddit threads were opened today. A real complaint study (20 threads from r/GradSchool, r/math, r/Anki, r/GetStudying) is still owed and belongs to Dino's design research.

### 2.4 The defensible wedge

**Rigor for hard, quantitative courses, proven by quotes.** Concretely, four things that reinforce each other:

1. **Verified quotes from the student's own course notes** on every correction, shown as the rendered formula or passage with its page. Code verifies it; the model cannot invent it.
2. **Math-grade input and output**: LaTeX in, KaTeX out, theorem conditions treated as separate claims, formulas transcribed and human-verified at ingestion.
3. **The loop**: a gap changes the review date of exactly that claim, and the week's budget follows.
4. **Published evaluation**: an eval card with checker precision and recall on a public fixture, later a delayed-test result with confidence intervals. Competitors publish adjectives.

What is not defensible: the AI student persona, card generation, the planner, reminders.

### 2.5 Target segment

**Primary ICP (proposal):** students in proof- or derivation-heavy courses, graduate and upper-year undergraduate, in mathematics, statistics, economics, quantitative finance and physics, at Ontario universities, who already run some manual review system (a spreadsheet, Anki, an error notebook). Dino is the archetype: five courses, a spaced-repetition sheet with a hand-picked interval ladder and an error log (planning dossier section 2).

Why this segment:
- Generic tools fail them visibly (math rendering, shallow cards), so the comparison is easy to win.
- They are 18 or older, which matches the launch restriction.
- Explaining a derivation is already how they are examined (orals, proofs, problem sets), so teach-back is not a foreign ritual.
- Cohorts are small and dense. One course champion can seed a whole class.

Honest limits: the segment is small, its size is [unverified], and education apps monetise poorly (RevenueCat median realized yearly value per payer $22.82, earlier research). This is a beachhead and a credibility base, not a venture-scale market. Expansion path, in order: upper-year STEM undergraduates in Ontario, then professional quantitative exams (actuarial, CFA) where material is owned by the candidate and willingness to pay is higher [unverified].

---

## 3. Product strategy

### 3.1 Positioning

> For students in math-heavy courses: explain it from memory, and get checked against your own course notes, with the exact passage shown for every correction. What you could not explain comes back sooner. The AI never gives you the answer first.

Words to avoid: "Feynman" as a brand (crowded), "first", "learn faster", "master", "boost grades", any learning-gain claim (`CLAUDE.md` rule, and the evidence does not exist). The working name "Space Repetition" reads as a misspelling of "spaced repetition" and is not searchable; choose a name before any public page (decision D16).

### 3.2 v1 feature set

**Must have (MVP for Dino and classmates)**

| Feature | Why it is in |
| --- | --- |
| Upload a PDF of course notes or a problem set; transcription to text plus LaTeX; page-by-page verification screen | Without correct source text there is nothing to quote |
| Concept and claim checklist proposed by the model, each claim anchored to a source block, approved or edited by the student | Claims are the reference the checker grades against |
| Typed teach-back with LaTeX and live preview, sources hidden; AI student asks one question per turn | The core act |
| Self-grade step before the report (student ticks which claims they think they covered) | Free calibration data, a fallback mode if the checker fails the gate, and the control condition for the pilot |
| Report: each claim with the student's words, label, correction, and the quoted block rendered with its page; "I disagree" button on every label | Trust and the data flywheel |
| Claim-level FSRS scheduling with manual override; visible before and after dates | The loop |
| Today queue across courses with estimated minutes; per-course weekly budget with the reason it changed | The planning value without timed blocks |
| Error log items (a problem that went wrong, with a note) attached to a concept | Matches how Dino already works |
| Accounts, per-user data isolation, delete my data and my files | Required for more than one user and for the rights position |
| "Checker unavailable" state that never changes a schedule | Protects the log |

**Later (pilot or launch)**

Daily email digest; voice input; photo of handwritten work; Anki and CSV import; course-pack sharing (claims only, never source text, or instructor-approved); per-user FSRS parameters; mobile-friendly card review; billing; an "AI student sits a quiz on what you taught" mode (only with code-sampled mastery, see 4.6).

**Cut**

Timed 30-minute calendar blocks; push notifications; the outside-source allowlist; the six-tab demo UI; speech synthesis of the student's replies; self-rated level as a planning multiplier after week one; Easy grades from teach-back (already excluded).

### 3.3 Pricing hypothesis (to test in the pilot, not to ship now)

Anchors: Synapsi $9.99 a month or $48 a year; RemNote Pro $8 and $18; Gizmo about $77 a year; Quizlet Plus $7.99 [unverified]; Google AI Plus $6.99 a month after a free year in Canada; ChatGPT study mode free.

- **Free through the pilot.** Charging before the checker is proven would spend trust that cannot be bought back.
- **Hypothesis A, term pass:** CA$29 per four-month term per student, all courses. Matches how students budget and avoids subscription-cancellation resentment (a recurring complaint about Gizmo).
- **Hypothesis B, freemium:** one course free forever, more courses CA$8 a month.
- **Hypothesis C, instructor or department pays:** a course licence where the instructor supplies the notes. Solves copyright and distribution at once; slow sales cycle.
- Estimated model cost at 3 sessions a week is under US$0.50 per user per month with the final-check design (section 4.5, my estimate, to be measured), so gross margin is not the constraint. Willingness to pay is.

### 3.4 Distribution

1. **Course-by-course seeding.** One champion per course builds the pack from their own copy of the notes; classmates join and upload their own copy. Start with Dino's five courses.
2. **Term timing.** Onboard in the first three weeks of a term and again two weeks before midterms. January 2027 is the first real window; September 2027 is the launch window.
3. **Instructor-approved packs.** Ask one or two instructors to permit their notes. The pitch to instructors: the AI never produces answers, it makes students explain. This is the strongest long-term channel and removes the rights problem.
4. **Evidence as content.** Publish the eval card and, later, the pilot result. Short honest write-ups travel in r/GradSchool, r/math and department Discords better than feature lists [unverified].
5. **Do not** recruit from any course where Cris or Dino has a teaching or grading role. It creates a power imbalance and would taint any study.
6. **Do not** compete on App Store search or programmatic SEO; the Feynman AI cluster owns that ground.

### 3.5 Evidence and trust strategy

| Stage | What may be said publicly | What backs it |
| --- | --- | --- |
| Now | "Experimental. We do not claim it improves grades." | Nothing yet |
| After gate G0 | "On our public test set the checker flagged X% of wrong claims and falsely flagged Y% of correct ones." | `eval/results/` committed, with dataset and commit hash |
| During pilot | Usage facts only (sessions, disputes, how often disputes were upheld) | Event log |
| After pilot | The delayed-test estimate with its confidence interval, whatever the direction | Pre-registered analysis plan |

In-product rules: every checker label shows its quote; every label can be disputed; disputes are reviewed weekly and upheld disputes become eval cases; the report says when the checker was unavailable; teach-back scheduling is labelled experimental until the pilot ends.

### 3.6 Privacy and copyright of course materials

Not legal advice. Positions below need a human to confirm.

- **Ownership.** York states "instructors at York University own the intellectual property to the teaching materials they create" and that uploading them to sites like Course Hero "would qualify as copyright infringement" (https://copyright.info.yorku.ca/students-reuse-of-teaching-materials-from-york-courses). The page predates or ignores AI tools. Dino's course codes look like York's [unverified, confirm with Dino].
- **Personal study.** Queen's vice-provost wrote on 19 May 2026 that downloading for personal study "is generally permitted under Canadian copyright law (fair dealing)", that uploading to a third-party platform is "governed by that platform's terms of service", and suggested instructors add an outline note that materials "should not be uploaded to third-party platforms" (https://www.queensu.ca/provost/teaching-and-learning/teaching-and-learning-communications/open-letters). So some course outlines will forbid exactly what the product needs.
- **Design consequences (proposals):**
  1. Uploaded sources are private to the uploading user. No search, sharing or reuse across users. Shared packs carry claims written by students and block references, never source text, unless an instructor approves.
  2. Quotes shown in reports are short excerpts to the same user who uploaded the source.
  3. No uploaded content is used to train anything. Model vendors are used under API terms that exclude training [verify each vendor's current terms before the pilot].
  4. One-click deletion of a source removes text, page images and derived blocks.
  5. An upload screen asks the student to confirm they are permitted to use the material for personal study and warns that some course outlines prohibit third-party uploads.
  6. **Immediate:** remove `content/source.md` and the source text inside `content/unit.json` from the repo and its history before any collaborator, public visibility or deployment; keep Dino's material in a gitignored private path as the plan originally stated (planning dossier section 4.3). Use an openly licensed text or team-written notes as the CI fixture [licence of any candidate text to be checked].
  7. Ask Dino's instructor for written permission to use the practice set in the pilot.
- **Privacy.** Collect the minimum: email, uploaded files, explanations, labels, review events. Weakness data is sensitive in practice; disclose that explanations and source excerpts are sent to US model APIs. Host the database in a Canadian region [provider choice open]. 18 and over only. Provide export and deletion. Earlier research flagged the OPC children's code and Bill C-36; their current status was not re-checked today [unverified].
- **Research ethics.** If the pilot's delayed test is meant for publication or is run under a university's name, a research ethics board review is needed before recruiting [unverified for the specific institutions; ask a supervisor]. A private product beta with informed consent does not produce publishable human-subject data.
- **Dino's spreadsheet template** is copyrighted by InfoInspired and licensed for personal use; do not copy its layout or formulas into the product (planning dossier section 2).

---

## 4. Technical design improvements

All changes respect the regulations: Go backend, Python only outside the request path, small PRs into `qa`, one unit per file boundary, an RPI folder per unit.

### 4.1 Target architecture

```
web/ (React, Vite, TS, KaTeX; built and embedded in the Go binary)
   |
cmd/api  ->  internal/app (composition root, handlers, auth middleware, SSE)
                |-- internal/teach     (session engine, student prompt, probe taxonomy)
                |-- internal/checker   (extract+match, verdict, adjudication)
                |-- internal/cite      (ONE normalization and verification function)   NEW
                |-- internal/srs       (claim-level FSRS, rollups, overrides)
                |-- internal/planner   (today queue, weekly budgets)
                |-- internal/pack      (claims proposal, anchoring, approval)          NEW
                |-- internal/ingest    (PDF pages -> blocks, verification state)       NEW
                |-- internal/store     (pgx implementations of every store interface)  NEW
                |-- internal/llm       (provider-neutral: Stream and JSON on both vendors, metering)
                |-- internal/auth      (magic link or OAuth, sessions)                 NEW
                |-- internal/feedback  (disputes, self-grades)                         NEW
cmd/worker   (ingestion and deferred checks; same binary, different entry)           NEW
cmd/evalcheck (uses internal/llm and internal/checker, no private adapters)
Postgres (single instance; append-only review_events; job table for the worker)
```

Reasons and trade-offs: one binary and one database keep operations within reach of two students; a Postgres job table avoids adding a queue service at the cost of throughput nobody needs yet; embedding the UI removes CORS and a second deploy at the cost of rebuilding the binary for UI changes.

### 4.2 Fixes for the audit findings

| Finding | Fix | Unit |
| --- | --- | --- |
| F1 citation dropped at Finish | New `internal/cite` with one `Normalize` and one `Verify`; checker and teach both call it; `finalize` downgrades contradicted to partial when a citation fails, same as the checker; regression test with a multi-line source | P0-CITE |
| F8 weak verification | Minimum quote length (proposal: 20 characters after normalization, or the whole block if the block is a display formula); block id must belong to an approved source of this user; locator derived by code from the block, not supplied by the model | P0-CITE, then M-CHECK2 |
| F2 leak through `probe_hint` | Remove free-text hints. Checker returns `{claim_id, gap_type}` only; code builds the probe from a fixed taxonomy plus the student's own sentence, or a human-approved cue label for an untouched claim (4.4) | M-PROBE |
| F2 leak suite is a fake | Real-model suite on the real unit with 30 adversarial probes; code check against a banned vocabulary derived from source terms not yet taught | P0-LLM, P0-GATE |
| F3 no Postgres teach store | `internal/store` implements `teach.Store`, writes `teach_sessions`, `teach_turns`, `taught_claims`, `claim_labels`; sessions survive restart | P0-STORE |
| F4 no loader or seed | `cmd/seed` loads a pack file into `sources`, `source_blocks`, `concepts`, `claims`, plus optional history | P0-INT |
| F5 UTC plan blocks | User time zone column; all day boundaries computed in it; timed blocks hidden in v1 | P0-TZ |
| F6 outage becomes Again | Report state `unchecked`; no `review_events` row; worker retries the check; UI says so. One final check at Finish replaces the 8-second wait | P0-OUTAGE |
| F7 no event id at Finish | Session id is the idempotency key, passed explicitly through the contract | P0-CONTRACT, P0-OUTAGE |
| F9 eval differs from production | `cmd/evalcheck` imports `internal/llm` and `internal/checker`; Gemini gets `JSON`, Anthropic gets `Stream`, so either vendor can fill either role | P0-LLM |
| F10 Gemini token cap and thinking | Set thinking configuration explicitly and raise the cap to about 200 tokens; verify on the real API [unverified today] | P0-LLM |
| F11 scheduler semantics | Resolved by claim-level state (4.3); never-reviewed claims appear in the queue as "new" | M-CLAIMSRS |
| F12 migrate lock, test rows left behind | Advisory lock in `cmd/migrate`; srs tests run in a throwaway schema per test run | P0-SRS |
| F13 process | Branch protection on `main` and `qa`, required PR and CI check; PR template with the unit name; one decisions file | P0-HYG |
| F14 small defects | Bounds check in `loadLabels`; server timeouts; robust JSON extraction through structured output | P0-LLM, P0-INT |
| TESTING.md B4 self-match | Change the pattern to real key shapes (for example `AIza[0-9A-Za-z_-]{30,}`) and exclude `docs/` | P0-HYG |

### 4.3 Data model changes (proposal for contract v2 and v3)

- `source_blocks(id, source_id, page, ordinal, kind [text|math|heading|figure], text_md, text_norm, verified_by, verified_at)`. Citations reference `block_id`.
- `source_pages(source_id, page, image_ref)` for display beside a quote.
- `claims` gain `kind [definition|statement|condition|method|result]`, `cue` (2 to 5 words, human-approved, safe to reveal), `anchor_block_ids`.
- `claim_states(user_id, claim_id, stability, difficulty, due_at, last_event_id)` replaces `concept_schedules` as the memory state. Concept due date is the earliest due required claim; concept retrievability is the mean. `concept_schedules` stays readable during migration.
- `review_events` gain `claim_id`, `session_id`, `self_grade`, `checker_model`, `checker_version`. Still append-only.
- `courses` (a pack: sources, concepts, claims, version) separate from `enrollments(user_id, course_id)`, so per-user state is keyed by user. Today concepts hang off a user's subject and `concept_schedules` has no user key (`migrations/0001_core.sql:104-105`), which blocks more than one user per pack.
- `disputes(id, user_id, claim_label_id, reason, resolution, resolved_by)`.
- `problems(id, user_id, concept_id, title, note, source_ref)` for the error log.
- `jobs(id, kind, payload, run_after, attempts, state)` for ingestion and deferred checks.
- `users` gain `email`, `time_zone`, `consent_version`, `deleted_at`. Every query filters by user; add an automated test that user A cannot read user B's rows through any endpoint.
- Contract additions flagged by the units and still open: pack loader type, `TeachStore`, `GetReport`, `GetPlan`, `DueItem.Reason`, richer `FurtherPlanning`, explicit event ids (code dossier section 3, `internal/domain`).

### 4.4 Teach engine and AI student

- **Probe taxonomy owned by code.** Types: ask for the definition, ask for the conditions, ask why a step holds, ask for an example, ask for a boundary case, ask to connect two taught statements, ask about an untouched topic by its cue. The student model receives the question type, the student's own sentence (or the approved cue), and nothing else. No model that has seen the source writes any text that reaches the student.
- **Per-turn matcher.** A cheap model receives the claim list as ids, kinds and cues (no source prose, no reference claim text beyond the cue) and the latest turn, and returns which claim ids were touched and a gap type. This is enough to pick the next probe. It is advisory and never graded.
- **Final check at Finish**, synchronous with a progress indicator, with a job fallback.
- **Honest framing.** The underlying model knows the subject. The product promise is "it only asks and never tells", which code can enforce, not "it knows nothing", which code cannot.
- Fix the sentence splitter for "e.g." and "i.e." (`internal/teach/student.go:134`).

### 4.5 Checker pipeline, cost and latency

Pipeline at Finish:
1. **Extract and match** (structured output): student statements mapped to claim ids.
2. **Verdict per matched pair** with `block_id` and quote, given only the blocks anchored to this concept plus neighbours. No vector search in v1; anchors do the retrieval.
3. **Code verification** through `internal/cite`.
4. **Adjudication of every `contradicted` label only**: a second pass by a different model or a second sample. Disagreement becomes `needs review`, shown as a suggestion, and does not change the schedule. A false accusation costs more trust than a missed error.
5. **Absent** is computed by code from untouched required claims, as today.

Cost per session, my estimates from list prices, to be replaced by `evalcheck` measurements:

| Design | Assumptions | Estimated cost |
| --- | --- | --- |
| Current: strong checker every turn, 8 turns | Sonnet 5.5 at $2 in, $10 out, $0.10 cache read per million tokens; about 5k cached source tokens, 2.5k fresh input, 1.2k output per turn | about US$0.14 plus the student |
| Proposed: cheap matcher per turn, one final check, adjudication on contradicted | Haiku 5.5 at $0.10 in, $0.50 out for 8 matcher calls; one Sonnet 5.5 call with about 8k input, 2k output; about 30% extra for adjudication | about US$0.05 |
| Student replies, 8 turns | A Flash-Lite or Haiku class model, about 700 tokens in, 60 out per turn | under US$0.002 |

At 3 sessions a week for 30 weeks that is about US$4.50 a year proposed versus about US$13 with per-turn checks. Prices: Anthropic's pricing page (https://platform.claude.com/docs/en/about-claude/pricing). Gemini prices come from a third-party page that says it checked Google's page on 25 Oct 2026 and that Gemini 3.6 and 3.8 Flash rates double on 1 Jan 2027 [unverified] (https://www.morphllm.com/gemini-api-pricing). If that holds, a Claude Haiku tier may be cheaper than a current Gemini Flash tier for the student role; the bake-off should include both and the decision is not mine to make.

Latency targets (proposals): student first token p95 under 1.5 s (unchanged); matcher under 2 s, non-blocking; final report p95 under 20 s with visible progress. The original "checker under 6 s per turn" target disappears with the design change.

Controls: per-user daily token budget; a hard monthly cap per environment; model ids and prices in config, never hardcoded; token metering on every call written to a `llm_calls` log table.

### 4.6 Eval harness

Layout: `eval/data/` (frozen, versioned datasets), `eval/results/` (one file per model per run with commit hash), `cmd/evalcheck` (runner). CI runs everything with the fake model on every PR; real-model runs are manual or nightly with keys in repository secrets and a spend cap.

| Suite | What it measures | Data | Proposed bar |
| --- | --- | --- | --- |
| Checker, human set | Claim-level precision and recall | 20 explanations, about 80 claim labels, labelled independently by Dino and a classmate, reconciled, frozen. Cohen's kappa at least 0.6 before freezing | False contradiction on truly supported claims at most 3%; wrong claims flagged at least 80%; absent claims caught at least 85% |
| Checker, perturbation set | Sensitivity to specific error types | At least 100 claims auto-perturbed (dropped condition, swapped parameter, wrong sign, wrong formula), 20% spot-checked by Dino | Flagged at least 85%; report per error type |
| Citation quality | Whether the quote supports the correction | Human rating of every correction in the human set | 100% pass code verification (by construction); at least 90% rated relevant |
| Stability | Same input, three runs | Human set | Label agreement at least 95% |
| Prompt injection | Student text trying to change labels | 10 attacks | 0 label changes |
| Coverage gradient ("mastery gradient" for the checker) | Explanation score falls as coverage falls | Synthetic explanations covering 100, 75, 50, 25, 0% of required claims for each concept | Monotone for every concept; Spearman at least 0.9 |
| Leak | Student states or names anything not taught and not an approved cue | 30 adversarial sessions on the real unit, real student model | 0 hard leaks. Note: 0 of 30 still allows a true rate up to about 10% at 95% confidence; grow the suite over time |
| Student mastery gradient | Only if the "AI student sits a quiz" mode is built: correctness sampled by code from taught-claim state | Simulated teaching profiles | Monotone gradient across profiles; no gain from wrong or irrelevant teaching |
| Ingestion fidelity | Formula and text accuracy | The 19 damaged regions listed in `content/damaged-formulas.md` plus a hand-corrected gold copy | At least 18 of 19 regions correct before human verification; 19 of 19 after |
| Cost and latency | Per session | All runs | Cost at most US$0.08 per session; latency as in 4.5 |

The four hackathon bars Cris approved stay as a smoke check. The table above replaces them as the product gate once confirmed. The gate rule from `rpi/GATE/PROMPT.md` stays: no prompt tuning or label edits to pass; tune only on a separate development split.

Bake-off candidates (proposal): Claude Sonnet 5.5 and Haiku 5.5; a Gemini Pro-class and a Flash-class model; each as checker, and the cheap ones as matcher and student. Report accuracy, cost and latency together. Budget cap proposal: US$50 for the whole bake-off.

### 4.7 Ingestion of math-heavy PDFs

Evidence: `pdftotext` flattens every display formula (`content/damaged-formulas.md`, visible in `content/source.md` page 2). A December 2025 benchmark of formula extraction on 100 synthetic pages scored Gemini 3 Pro 9.75, PaddleOCR-VL 9.65, Mathpix 9.64, MinerU 2.5 9.17, Gemini 2.5 Flash 8.78, Mistral OCR 8.66, PyMuPDF4LLM 6.67 on a 0 to 10 judged scale (https://arxiv.org/abs/2512.09874). Docling does not appear in that table, so its math quality remains [unverified]. Synthetic pages are easier than scanned or handwritten notes.

Design (proposal):
1. Split the PDF into pages and render page images (Go, in the worker). Store the images.
2. Transcribe each page with a vision model into Markdown with LaTeX, as ordered blocks.
3. Run a second, different model on pages that contain display math; blocks where the two disagree are flagged.
4. **Verification screen:** page image on the left, rendered blocks on the right; the student confirms or fixes each flagged block. Only verified blocks can be cited. This is also legitimate study time.
5. Whether the worker stays Python (Docling or MinerU, allowed by the existing exception) or becomes Go calling a vision API is decided by a one-day spike on Dino's 13 pages against the gold copy. A Go-only path removes the second language [unverified that Go PDF splitting and rendering is adequate].
6. Handwritten notes and slides with figures are out of scope until the pilot asks for them. Ask Dino what form his five courses' materials take.

Cost is small: 13 pages on a Pro-class vision model is well under US$1 [my estimate].

### 4.8 UI direction

Regulation 6 and 7 say UI is deferred and must be polished when its phase starts. The only logged exception was the hackathon demo. An MVP for real users needs a real UI, so the regulation must be changed in writing first (decision D8), and Dino's design note on three dashboards, owed since the plan, becomes the first input.

Four screens, desktop web first because math typing needs a keyboard:

1. **Today.** Due items across courses with minutes, grouped by course; the week's budget per course with one line explaining any change ("MATH 6602: 3.5 h, up from 2 h, because 3 gaps opened on Monday").
2. **Teach.** One editor with LaTeX preview, the AI student's single question above it, no sources visible, a timer, Finish. Microphone is a button that fills the same editor.
3. **Report.** Self-grade first, then the claim table: your words, label, the quoted block rendered with its page thumbnail, dispute button, and the date change per claim. An "unchecked" banner when the checker was unavailable.
4. **Course.** Sources and their verification status, the concept and claim checklist, recall and explanation shown separately per concept, the error log.

Design principles: one accent colour, labels in words plus colour, formulas always rendered, every number explains itself on hover, no streaks, lives or leaderboards. Playwright covers the full typed loop against the fake model.

### 4.9 Operations

Single small instance and managed Postgres in a Canadian region [provider and region availability unverified]; daily backups with a restore test; secrets in the host's secret store; structured logs with no explanation text at info level; error alerting to Cris; rate limits per user; a status flag that disables scheduling changes if the checker error rate exceeds a threshold.

---

## 5. Roadmap

Effort and dates are unvalidated estimates. Team capacity per week is unknown [ask Cris and Dino]. Each unit gets `rpi/<unit>/PROMPT.md`, `research.md`, `plan.md`, `impl-log.md`, a branch from `qa`, small PRs, and a file boundary. Only the contract units may touch `internal/domain/`, `api/` or `migrations/`, and Cris owns them.

### Phase 0: stabilize and prove the thesis (proposed 10 Nov to 23 Nov 2026)

**Goal:** the loop runs end to end on one unit with real models, and gate G0 has real numbers.
**Non-goals:** UI polish, upload, accounts, digest, planner changes beyond the time-zone fix.

Proposed `boundaries.json` entries:

```json
{
  "P0-HYG":      {"branch": "chore/repo-hygiene",      "allow": [".github/", ".gitignore", "CLAUDE.md", "docs/", ".claude/", "content/", "rpi/P0-HYG/"]},
  "P0-CONTRACT": {"branch": "feature/contract-v2",     "allow": ["internal/domain/", "api/", "migrations/", "internal/fakes/", "rpi/P0-CONTRACT/"]},
  "P0-CITE":     {"branch": "fix/citation-normalize",  "allow": ["internal/cite/", "internal/checker/", "internal/teach/engine.go", "internal/teach/engine_test.go", "rpi/P0-CITE/"]},
  "P0-LLM":      {"branch": "feature/llm-parity",      "allow": ["internal/llm/", "cmd/evalcheck/", ".env.example", "rpi/P0-LLM/"]},
  "P0-EVALDATA": {"branch": "feature/eval-data",       "allow": ["eval/data/", "scripts/check_labels.py", "rpi/P0-EVALDATA/"]},
  "P0-STORE":    {"branch": "feature/pg-store",        "allow": ["internal/store/", "rpi/P0-STORE/"]},
  "P0-OUTAGE":   {"branch": "fix/checker-outage",      "allow": ["internal/teach/", "rpi/P0-OUTAGE/"]},
  "P0-SRS":      {"branch": "fix/srs-idempotency",     "allow": ["internal/srs/", "cmd/migrate/", "rpi/P0-SRS/"]},
  "P0-TZ":       {"branch": "fix/planner-timezone",    "allow": ["internal/planner/", "rpi/P0-TZ/"]},
  "P0-INT":      {"branch": "feature/integration",     "allow": ["cmd/api/", "cmd/seed/", "internal/app/", "Makefile", "compose.yaml", "go.mod", "go.sum", "rpi/INT/"]},
  "P0-GATE":     {"branch": "feature/gate-results",    "allow": ["eval/results/", "rpi/GATE/"]},
  "P0-SLICE":    {"branch": "feature/dogfood-page",    "allow": ["web/", "rpi/P0-SLICE/"]}
}
```

Ordering: P0-HYG and P0-CONTRACT first. P0-CITE before P0-OUTAGE (both touch `internal/teach/engine.go`, so they run in series). P0-LLM, P0-EVALDATA, P0-STORE, P0-SRS, P0-TZ in parallel. Then P0-INT, P0-GATE, P0-SLICE.

| Unit | Work | Owner | Acceptance |
| --- | --- | --- | --- |
| P0-HYG | Branch protection on `main` and `qa`; CI (build, vet, test with Postgres, race); PR template; `docs/DECISIONS.md` as the one decision log; update `CLAUDE.md` (deadline line, product phase); fix `docs/TESTING.md` B4 and the gate file name; move course material to a gitignored path and add a licensed or team-written fixture; history purge is a separate step Cris runs by hand | Cris (Claude drafts) | A direct push to `qa` is rejected; CI green on a PR; `git grep` for the practice-set text returns nothing on `qa` |
| P0-CONTRACT | Contract v2: pack loader type, `TeachStore`, `GetReport`, explicit event ids, `DueItem.Reason`, report status `unchecked`, `source_blocks`, user time zone | Cris | `go build ./...` green; fakes updated; OpenAPI validated by a real validator |
| P0-CITE | F1 and F8 | Claude, Cris reviews | New test: multi-line source, quote across the break survives Finish; one-word quote rejected |
| P0-LLM | F9, F10, F14; structured output; metering; first real calls | Claude with Cris's keys | `evalcheck --model fake` and one real model per vendor run through `internal/llm`; cost per session printed |
| P0-EVALDATA | 20 explanations at claim level, two independent label files, kappa script, frozen final labels, perturbation set, dev and test split | Dino and a classmate label; Claude builds tooling; Cris approves the concept list first | Kappa at least 0.6; files frozen with a hash recorded in `docs/DECISIONS.md` |
| P0-STORE | Postgres `teach.Store`, session persistence, approved-source filter | Claude | Restart mid-session and Finish still works; `claim_labels` rows exist after Finish |
| P0-OUTAGE | F6, F7; final check at Finish | Claude | With the checker failing, the report is `unchecked` and `review_events` gains no row |
| P0-SRS | F7 at the DB level, F12 | Claude | Finish retried after restart writes one event; test run leaves no rows in the dev DB |
| P0-TZ | F5 | Claude | Blocks for a Toronto user fall between the configured local hours across a DST change |
| P0-INT | Composition root, seed loader, handlers for teach, report, due, review, override; one automated full-loop test with fakes, one run with real models | Cris with Claude | The INT prompt's assertion list passes (`rpi/INT/PROMPT.md`) |
| P0-GATE | Run all suites in 4.6 that have data; write `eval/results/gate.md` | Cris runs, Dino reads, both decide | Numbers reported against bars with every failed example in full |
| P0-SLICE | One plain page: pick concept, type explanation, see the report | Claude, Dino reviews | Dino completes 3 real sessions on the practice-set unit |

**Gate G0 (proposals to confirm):** every bar in the 4.6 table for the checker human set, citation quality, stability, injection, coverage gradient, leak, and cost; plus the full-loop test green with real models; plus Dino's 3 sessions done with his written notes on what felt wrong.

**If G0 fails:** one iteration on the development split (narrower claims, different checker model). If it still fails, ship self-graded teach-back (reveal the claims with their quotes, the student grades themself) with checker labels shown as suggestions that do not change the schedule, and re-evaluate the product thesis before Phase 1.

### Phase 1: MVP for Dino and classmates (proposed 24 Nov 2026 to 10 Jan 2027)

**Goal:** five or more students can upload their own notes for a quantitative course, teach back, and trust the schedule, without Cris touching the database.
**Non-goals:** billing, mobile app, digest, pack sharing, voice, per-user FSRS fitting, any public marketing.

```json
{
  "M-CONTRACT3": {"branch": "feature/contract-v3",   "allow": ["internal/domain/", "api/", "migrations/", "internal/fakes/", "rpi/M-CONTRACT3/"]},
  "M-INGEST":    {"branch": "feature/ingest-math",   "allow": ["internal/ingest/", "cmd/worker/", "scripts/parse_pdf.py", "eval/ingest/", "rpi/M-INGEST/"]},
  "M-PACK":      {"branch": "feature/pack-builder",  "allow": ["internal/pack/", "rpi/M-PACK/"]},
  "M-CHECK2":    {"branch": "feature/checker-final", "allow": ["internal/checker/", "internal/cite/", "rpi/M-CHECK2/"]},
  "M-PROBE":     {"branch": "feature/probe-taxonomy","allow": ["internal/teach/", "rpi/M-PROBE/"]},
  "M-CLAIMSRS":  {"branch": "feature/claim-schedule","allow": ["internal/srs/", "rpi/M-CLAIMSRS/"]},
  "M-TODAY":     {"branch": "feature/today-queue",   "allow": ["internal/planner/", "rpi/M-TODAY/"]},
  "M-AUTH":      {"branch": "feature/auth",          "allow": ["internal/auth/", "rpi/M-AUTH/"]},
  "M-FEEDBACK":  {"branch": "feature/disputes",      "allow": ["internal/feedback/", "rpi/M-FEEDBACK/"]},
  "M-STORE2":    {"branch": "feature/pg-store-v3",   "allow": ["internal/store/", "rpi/M-STORE2/"]},
  "M-API":       {"branch": "feature/api-v1",        "allow": ["cmd/api/", "internal/app/", "Makefile", "go.mod", "go.sum", "rpi/M-API/"]},
  "M-WEB":       {"branch": "feature/web-app",       "allow": ["web/", "rpi/M-WEB/"]},
  "M-DEPLOY":    {"branch": "chore/deploy",          "allow": ["deploy/", "Dockerfile", ".github/workflows/", "rpi/M-DEPLOY/"]},
  "M-QA":        {"branch": "chore/qa-mvp",          "allow": ["qa/", "rpi/M-QA/"]}
}
```

| Unit | Work | Owner | Acceptance |
| --- | --- | --- | --- |
| M-CONTRACT3 | Section 4.3 model: courses and enrollments, `claim_states`, disputes, problems, jobs, user fields | Cris | Migration applies on a copy of the Phase 0 database with no data loss |
| M-INGEST | 4.7 pipeline and spike | Claude; Dino supplies gold pages | Ingestion fidelity bar met; a 13-page PDF is ready to verify in under 5 minutes |
| M-PACK | Claim proposal with anchors, kinds and cues; approval flow | Claude; Dino judges quality | For 3 real units, at least 80% of proposed required claims accepted without edit by Dino |
| M-CHECK2 | Two-stage check, block citations, adjudication | Claude | G0 bars still met on the frozen test split; no regression |
| M-PROBE | Taxonomy probes, matcher, no free-text hints | Claude | Leak suite 0 of 30; Dino rates at least 70% of probes "useful" in 5 sessions |
| M-CLAIMSRS | Claim-level state and rollups | Claude | A session with one missing claim moves only that claim earlier; benchmark still under 20 ms per 1,000 items |
| M-TODAY | Today queue, weekly budget with reasons | Claude | Deterministic; reasons shown for every change |
| M-AUTH | Email magic link or Google sign-in, sessions | Cris | Isolation test: user A cannot read user B through any endpoint |
| M-FEEDBACK | Self-grade capture, disputes, weekly review export | Claude | Every dispute appears in an export Dino can read |
| M-STORE2, M-API | Stores and all handlers for the above | Cris with Claude | OpenAPI and handlers match (contract test) |
| M-WEB | Four screens from 4.8 | Claude builds, Dino owns design review | Playwright full loop green; Dino signs off on each screen |
| M-DEPLOY | Hosting, backups, secrets, budgets | Cris | Restore test passes; spend cap proven by a test |
| M-QA | Test plan run by Dino on the deployed build | Dino | Report with defects; no severity-high defect open |

**Gate G1 (proposals):**
- Dino completes at least 12 sessions across at least 3 weeks on at least 2 courses.
- At least 4 other students each complete at least 4 sessions within 2 weeks of onboarding.
- Disputes on at most 10% of corrections, and at least 70% of reviewed disputes resolved in the checker's favour.
- Ingestion: at most 10 minutes of human fixing per 10 pages.
- Zero cross-user data exposures in tests; zero schedule changes from unchecked reports.
- Median model cost per session at most US$0.08.
- Written permission from at least one instructor, or a recorded decision to proceed on the personal-study position.

### Phase 2: pilot (proposed 11 Jan to 30 Apr 2027)

**Goal:** learn whether students keep using it and whether there is any measurable effect, with honest statistics.
**Non-goals:** paid acquisition, new domains, mobile app.

```json
{
  "PL-STUDY":   {"branch": "feature/study-arms",       "allow": ["internal/study/", "rpi/PL-STUDY/"]},
  "PL-METRICS": {"branch": "feature/metrics-export",   "allow": ["internal/metrics/", "cmd/export/", "rpi/PL-METRICS/"]},
  "PL-DIGEST":  {"branch": "feature/digest",           "allow": ["internal/digest/", "rpi/U6/"]},
  "PL-PRIVACY": {"branch": "feature/privacy-controls", "allow": ["internal/privacy/", "docs/legal/", "rpi/PL-PRIVACY/"]},
  "PL-IMPORT":  {"branch": "feature/import",           "allow": ["internal/importer/", "rpi/PL-IMPORT/"]},
  "PL-WEB2":    {"branch": "feature/web-pilot",        "allow": ["web/", "rpi/PL-WEB2/"]},
  "PL-EVAL2":   {"branch": "feature/eval-from-disputes","allow": ["eval/", "cmd/evalcheck/", "rpi/PL-EVAL2/"]}
}
```

| Unit | Work | Owner |
| --- | --- | --- |
| PL-STUDY | Within-student randomization of topics to teach-back plus spaced recall, spaced recall only, or self-graded teach-back; consent flow; pre-registered analysis plan | Cris (design with a supervisor's advice), Dino (test items with instructor input) |
| PL-METRICS | Retention, sessions per week, dispute rates, cost; export for analysis | Claude |
| PL-DIGEST | One daily email, suppressed after study, sent in the user's time zone, with an A/B switch | Claude |
| PL-PRIVACY | Export, deletion, policy and consent texts | Cris; a human with legal knowledge reviews |
| PL-IMPORT | CSV and Anki import; error-log import from a sheet | Claude |
| PL-WEB2 | Onboarding, empty states, pilot feedback form | Claude, Dino reviews |
| PL-EVAL2 | Upheld disputes become eval cases; monthly re-run of the gate | Claude, Dino adjudicates |

**Gate G2 (proposals):**
- At least 20 students active in week 4; week-4 retention at least 35% of those onboarded.
- Median at least 2 sessions a week among active users.
- At least 40% of active users answer "very disappointed" to "how would you feel if you could no longer use this".
- Checker gate re-run on the grown eval set still meets G0 bars.
- Delayed unaided test result reported with a 95% interval. Bar: the interval is published whatever it shows; if the estimate is negative and the interval excludes zero, stop and redesign.
- Model cost at most US$1.50 per active user per month.
- Willingness to pay: at least 25% of active users say they would pay CA$8 a month or CA$29 a term, or at least 10 pre-orders.
- If fewer than three of the first five bars are met, do not proceed to launch; decide between a research write-up and a pivot.

### Phase 3: market launch (proposed May to Sept 2027, live for the fall term)

**Goal:** a paid product for the fall 2027 term in Ontario, with published evidence.
**Non-goals:** K-12, users under 18, native mobile apps, enterprise features.

```json
{
  "L-BILLING":  {"branch": "feature/billing",          "allow": ["internal/billing/", "rpi/L-BILLING/"]},
  "L-SHARE":    {"branch": "feature/pack-sharing",     "allow": ["internal/share/", "rpi/L-SHARE/"]},
  "L-SITE":     {"branch": "feature/landing-site",     "allow": ["site/", "rpi/L-SITE/"]},
  "L-WEB3":     {"branch": "feature/web-launch",       "allow": ["web/", "rpi/L-WEB3/"]},
  "L-FSRSOPT":  {"branch": "feature/fsrs-params",      "allow": ["internal/srs/", "rpi/L-FSRSOPT/"]},
  "L-LOAD":     {"branch": "chore/load-and-security",  "allow": ["qa/", "rpi/L-LOAD/"]},
  "L-INSTRUCT": {"branch": "feature/instructor-packs", "allow": ["internal/instructor/", "rpi/L-INSTRUCT/"]}
}
```

| Unit | Work | Owner |
| --- | --- | --- |
| L-BILLING | Term pass and subscription, tax handling, cancellation that is one click | Cris |
| L-SHARE | Claim-only pack sharing; instructor-approved full packs | Claude, Cris reviews rights logic |
| L-INSTRUCT | Instructor upload and approval flow | Claude |
| L-SITE | Landing page with the eval card and pilot results, no learning claims beyond what was measured | Dino (copy and design), Claude builds |
| L-WEB3 | Responsive review on phones, accessibility pass | Claude, Dino reviews |
| L-FSRSOPT | Per-user parameters only if the log shows benefit on held-out reviews | Claude |
| L-LOAD | Load test at 500 concurrent users on non-LLM paths, security review, restore drill | Cris, external reviewer if possible |

**Gate G3 (proposals):** 200 sign-ups in the first six weeks of term; 15% of week-4 actives paying; p95 non-LLM read under 50 ms at 500 concurrent users; refund rate under 5%; zero privacy incidents; checker gate still met on the current eval set.

---

## 6. Decisions only Cris and Dino can make

| # | Decision | Options | My recommendation and why |
| --- | --- | --- | --- |
| D1 | What is this through April 2027 | Validated tool and evidence; startup now; research study; portfolio piece | **Validated tool with a pilot, business decision at G2.** The thesis is unproven, the market leader is free, and both of you are full-time students. A measured result is valuable under every outcome |
| D2 | Target segment | Quantitative graduate and upper-year courses in Ontario; all university students; exam-prep markets | **Quantitative courses, Ontario first.** It is where generic tools fail visibly and where you have a user |
| D3 | Input | Typed LaTeX first, voice optional; voice first (current) | **Typed first.** The first user's material is formulas; speech is the plan's own High risk |
| D4 | Memory unit | Claim-level state with concept rollup; concept-level (current) | **Claim-level.** Matches how FSRS models memory and makes "this gap comes back sooner" literally true |
| D5 | Checker timing | One final check plus cheap per-turn matcher; strong check every turn (current) | **Final check.** About a third of the cost, removes the Finish race, simpler to evaluate |
| D6 | Probe hints | Code taxonomy plus approved cue labels; free-text hints from the checker (current); no topic cues at all | **Taxonomy plus cues.** Closes the leak while still letting the student ask about an untouched topic |
| D7 | Course material | Purge from the repo and history, keep private, seek instructor permission; leave as is in a private repo | **Purge and seek permission.** York's stated position makes the current state a liability the day anyone else gets access |
| D8 | Regulations | Lift "UI deferred" for the MVP; keep it. Also: keep, drop or relax the Jev rule; confirm the three additions Claude made (branch flow, artifact list, stack list); pick one regulations file | **Lift the UI deferral with a logged change; make Jev optional with a logged manual split when it is not run; confirm the additions; keep `claude/regulations.md` and delete the duplicate.** A rule nobody follows weakens the rules that matter |
| D9 | Gate bars | Adopt the 4.6 table; keep the four hackathon bars | **Adopt 4.6, keep the old bars as a smoke check.** The old sample sizes cannot support a product decision |
| D10 | Labellers | Dino plus a classmate; Cris and Dino (current) | **Dino plus a classmate.** Labels need domain knowledge |
| D11 | Models and budget | Bake-off across both vendors for all three roles with a US$50 cap; fixed assignment (Gemini student, Claude checker) | **Bake-off decides.** I am a Claude model and should not pick |
| D12 | Pilot form | Product beta with consent; formal study with ethics review | **Start as a beta; ask a supervisor in November whether ethics review is needed if you want to publish.** The lead time for review is long and the January window is fixed |
| D13 | Planner scope in v1 | Today queue and weekly budgets; timed calendar blocks | **Today queue.** Timed blocks are unvalidated and absent from Dino's real workflow |
| D14 | Outside-source allowlist | Drop from v1; pursue LeetCode-style packs | **Drop.** Licence risk and off-wedge |
| D15 | Pricing test | Free through pilot then term pass; freemium; charge now | **Free through pilot, test the term pass at G2** |
| D16 | Name | Rename before any public page; keep "Space Repetition" | **Rename.** The current name reads as a typo and cannot be searched |
| D17 | Hosting and data location | Canadian region for the database, US model APIs disclosed; anything else | **Canadian region, disclosed US processing** |
| D18 | Capacity | Hours per week each of you commits through January | No recommendation. The roadmap dates are meaningless until this is stated |
| D19 | Self-grade step | Always on; optional; off | **Always on.** It is the fallback product, the control arm and free calibration data |
| D20 | U1 concept list | Approve the 8 draft concepts; rewrite claims as definitions, statements, conditions and methods | **Rewrite with Dino.** Numeric worked answers make poor teach-back targets |

---

## 7. Risks and mitigations

| Risk | Likelihood | Impact | Mitigation |
| --- | --- | --- | --- |
| Checker cannot reach the precision bar on graduate math | Medium to high (never tested) | Fatal to the wedge | G0 first; adjudication on contradicted; self-grade fallback; narrow claim kinds; stop rule in Phase 0 |
| Students will not do effortful teach-back (Khanmigo-style 15% engagement) | High | Fatal to retention | Short sessions (5 to 8 minutes, one concept); teach-back announced in the Today queue; measure at G1 and G2 before investing in launch |
| Google or OpenAI adds spaced teach-back with grounding | Medium within 12 months | High | Stay narrow on rigor and math; publish evidence; instructor-approved packs they cannot easily copy |
| Synapsi or AeraWise ships the same loop first | High | Medium | Do not race on features; win on verified quotes, math fidelity and published evaluation |
| Copyright complaint from an instructor or university | Medium | High | Section 3.6 design; purge the repo; instructor permission; upload confirmation; fast takedown path |
| Wrong correction shown with a real quote (quote is real, reasoning is wrong) | Medium | High for trust | Human-rated citation relevance bar; dispute button; adjudication; "needs review" state |
| Ingestion errors in formulas become "verified" quotes | Medium | High | Two-model diff, human verification screen, only verified blocks citable, page image shown beside every quote |
| Teach-back to FSRS mapping distorts schedules | Medium | Medium | Separate event type; self-grade recorded alongside; compare predicted recall with later card reviews during the pilot; manual override stays |
| Vendor outage or price change (Gemini Flash prices reportedly doubling in January [unverified]) | Medium | Medium | Provider-neutral `internal/llm`; either vendor can fill any role; `unchecked` state; budget caps |
| Prompt injection through student text or uploaded PDFs | Medium | Medium | Untrusted fencing already in place; add the injection suite; treat source text as data in the checker prompt as well |
| Privacy incident (cross-user leak of explanations or files) | Low to medium | High | Isolation tests in CI; per-user scoping in every store method; minimal logging |
| Team capacity: two students with full course loads and other commitments | High | High | State hours (D18); Phase 0 is sized for part-time work; cut order: digest, import, pack sharing, voice |
| Process erosion (PRs skipped, docs diverge) as on day one | High | Medium | Branch protection; CI required; `docs/DECISIONS.md` as the single log, mirrored to the Claude Project after each change |
| Pilot is underpowered and shows nothing | High | Medium | Within-student design; report intervals; pre-register; treat usage and trust metrics as the primary pilot outcomes |
| Single labelling expert (Dino) is a bottleneck | Medium | Medium | Recruit one classmate now; grow the eval set from upheld disputes |
| My own bias toward Claude models | Present | Low to medium | Bake-off with both vendors; humans decide |

---

## 8. What to do this week (proposal)

1. Cris and Dino read sections 0 and 6 and answer D1 to D7 and D18 in `docs/DECISIONS.md`.
2. Cris: branch protection, CI, purge the course material from tracked files (P0-HYG).
3. Dino: recruit one classmate as second labeller; rewrite the claim list for 4 concepts with Cris; write the owed design note on three dashboards.
4. Claude sessions under Cris: P0-CITE, then P0-LLM and P0-STORE in parallel worktrees.
5. Cris: add keys locally, make the first real call to each vendor, record model ids in `.env.example` by name.
6. Cris: ask a supervisor whether a delayed-test pilot needs ethics review.

---

## 9. Sources used for the market update

Fetched today unless noted.

- AeraWise home page: https://aerawise.com/
- Synapsi App Store listing: https://apps.apple.com/app/synapsi-ai-study-notes/id6780159251 (site synapsi.guru not reachable from this session)
- Google, study notebooks (25 Jun 2026): https://blog.google/innovation-and-ai/products/gemini-app/gemini-study-notebooks/
- Google, new Gemini Notebook study tools (15 Sep 2026): https://blog.google/innovation-and-ai/products/gemini-notebook/new-study-tools-september-2026/
- Google Canada, free year of AI Plus for post-secondary students (28 Oct 2026): https://blog.google/intl/en-ca/products/explore-get-answers/eligible-canadian-post-secondary-students-can-get-one-year-of-gemini-on-us/
- ChatGPT learning what's new: https://learn.chatgpt.com/docs/whats-new
- Claude learning mode coverage (search result only, not re-read): https://www.engadget.com/ai/anthropic-brings-claudes-learning-mode-to-regular-users-and-devs-170018471/ and https://www.tomsguide.com/ai/claudes-new-learning-modes-take-on-chatgpts-study-mode-heres-what-they-do
- RemNote pricing (third party): https://toolradar.com/tools/remnote/pricing
- Knowt pricing (third party): https://toolradar.com/tools/knowt/pricing
- Gizmo versus Knowt review: https://www.techraisal.com/blog/gizmo-ai-vs-knowt-i-used-both-for-a-full-semester-and-who-actually-wins-in-2026/
- Gizmo Series A: https://doolpa.com/news/gizmo-22m-series-a-gamified-learning-april-2026
- Spaced repetition app updates (competitor blog): https://www.mindomax.com/spaced-repetition-apps-updates-2026
- Anki team update, June 2026: https://forums.ankiweb.net/t/updates-from-the-anki-team-june-2026/70120
- Feynman AI: https://feynman-ai.com/ and https://getfeynman.app/
- Alice pricing: https://carrot.alice.tech/en-gb/pricing/
- Reddit themes (vendor blog): https://scholarly.so/blog/best-ai-for-studying-reddit-2026
- Formula extraction benchmark: https://arxiv.org/abs/2512.09874
- Anthropic API pricing: https://platform.claude.com/docs/en/about-claude/pricing
- Gemini API pricing (third party): https://www.morphllm.com/gemini-api-pricing
- York University, students' reuse of teaching materials: https://copyright.info.yorku.ca/students-reuse-of-teaching-materials-from-york-courses
- Queen's University open letter on course materials and AI tools (19 May 2026): https://www.queensu.ca/provost/teaching-and-learning/teaching-and-learning-communications/open-letters

Not done today and still owed: hands-on testing of Synapsi, AeraWise and Gemini study notebooks on one of Dino's units (the most useful single piece of market research left, about two hours); direct reading of Reddit threads; current status of Canadian privacy bills; segment size for quantitative students in Ontario; Docling and MinerU on real course notes.
