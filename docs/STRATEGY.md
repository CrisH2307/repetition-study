# Product strategy (proposal)

Status: PROPOSAL from the Fable 5.2 strategy of 2026-11-09. Segment, positioning,
features and pricing are pending Cris and Dino (D1, D2, D3, D15, D16 in
docs/DECISIONS.md). Market facts come from public pages read on 2026-11-09;
nothing was tested hands-on. Items marked [unverified] could not be confirmed.

## Positioning

For students in math-heavy courses: explain it from memory, get checked against
your own course notes, and see the exact passage for every correction. What you
could not explain comes back sooner. The AI never gives you the answer first.

Avoid: "Feynman" as a brand (crowded), "first", "learn faster", "master", any
learning-gain claim. The working name reads as a typo of "spaced repetition"
(D16).

## Wedge

Rigor for hard, quantitative courses, proven by quotes:

1. Verified quotes from the student's own notes on every correction, checked by code.
2. Math-grade input and output: LaTeX in, KaTeX out, theorem conditions as separate claims.
3. The loop: a gap moves the review date of exactly that item, and the week's budget follows.
4. Published evaluation: checker precision and recall on a public fixture.

Not defensible: the AI student persona, card generation, the planner, reminders.

## Segment (D2)

Graduate and upper-year students in proof- or derivation-heavy courses
(mathematics, statistics, economics, quantitative finance, physics) at Ontario
universities, who already run a manual review system. Dino is the archetype: five
courses, a spaced-repetition sheet with a hand-picked interval ladder and an
error log. All are 18 or older. Segment size is [unverified]; this is a
beachhead, not a large market. Later: upper-year STEM undergraduates, then
professional quantitative exams [unverified willingness to pay].

## v1 features versus later

| v1 (MVP for Dino and classmates) | Later | Cut |
| --- | --- | --- |
| PDF upload, transcription to text plus LaTeX, page verification screen | Daily digest | Timed 30-minute calendar blocks |
| Claim checklist anchored to source blocks, approved by the student | Voice input (D3) | Push notifications |
| Typed teach-back with live LaTeX preview, sources hidden, one question per turn | Photo of handwritten work | Outside-source allowlist (D14) |
| Self-grade before the report (D19) | Anki and CSV import | Six-tab demo UI |
| Report with words, label, rendered quote and page, dispute button | Claim-only pack sharing | Speech synthesis of replies |
| Claim-level FSRS with manual override (D4) | Per-user FSRS parameters | Streaks, lives, leaderboards |
| Today queue across courses with weekly budgets and reasons (D13) | Billing | |
| Error-log items attached to a concept | Mobile card review | |
| Accounts, per-user isolation, delete my data | | |
| "Checker unavailable" state that never changes a schedule (D23) | | |

## Pricing hypothesis (D15, test in the pilot)

Free through the pilot. Then test:
- A: term pass, CA$29 per four-month term, all courses.
- B: freemium, one course free, more courses CA$8 a month.
- C: instructor or department licence where the instructor supplies notes.

Estimated model cost under US$0.50 per user per month with a final-check design
(Claude estimate from list prices, to be measured by P0-LLM). Willingness to pay
is the constraint, not margin.

## Distribution

1. Course-by-course seeding: one champion per course; classmates upload their own copy.
2. Term timing: first three weeks of a term and two weeks before midterms. January 2027 first; September 2027 launch.
3. Instructor-approved packs: the pitch is that the AI makes students explain and never answers.
4. Evidence as content: eval card first, pilot result later, in r/GradSchool, r/math and department Discords [unverified reach].
5. Never recruit from a course where Cris or Dino teaches or grades.
6. Do not compete on App Store search or SEO.

## Evidence and trust

| Stage | What may be said | Backing |
| --- | --- | --- |
| Now | "Experimental. We do not claim it improves grades." | Nothing yet |
| After G0 | "On our public test set the checker flagged X% of wrong claims and falsely flagged Y% of correct ones." | `eval/results/` with dataset and commit hash |
| Pilot | Usage facts only | Event log |
| After pilot | Delayed-test estimate with its interval, whatever the direction | Pre-registered plan |

In product: every label shows its quote, every label can be disputed, upheld
disputes become eval cases, the report says when the checker was unavailable.

## Privacy and copyright (not legal advice; a human must confirm)

- York states instructors own their teaching materials and that uploading them to
  sites like Course Hero "would qualify as copyright infringement"
  (https://copyright.info.yorku.ca/students-reuse-of-teaching-materials-from-york-courses).
  Whether Dino's courses are York courses is [unverified].
- Queen's (19 May 2026): personal-study downloading is generally fair dealing;
  third-party uploads follow the platform's terms; instructors may forbid uploads
  (https://www.queensu.ca/provost/teaching-and-learning/teaching-and-learning-communications/open-letters).
- Proposed rules: uploads private to the uploader; no sharing of source text
  across users; no training on uploads; one-click deletion of text, images and
  derived blocks; upload screen asks the student to confirm personal-study use.
- Immediate (D7): remove the practice set from tracked files and history; ask the
  instructor for permission; use a licensed or team-written CI fixture.
- Privacy: minimum data; disclose US model processing; Canadian database region
  (D17); 18 and over; export and deletion. Status of Canadian privacy bills not
  re-checked [unverified].
- A pilot meant for publication likely needs ethics review [unverified]; ask a
  supervisor (D12).
- Dino's sheet template is licensed for personal use only; do not copy it.

## Competitors (public pages, 2026-11-09)

| Product | Overlap | Price | Not found | URL |
| --- | --- | --- | --- | --- |
| Synapsi | AI-judged Feynman teach-back against the source, FSRS | US$9.99/mo, US$48/yr | Verified quotes, math rendering, cross-course planning | https://apps.apple.com/app/synapsi-ai-study-notes/id6780159251 |
| AeraWise | "Teach Lab" AI student, Feynman reports, planner, spaced repetition | Free | Teach-back changing the schedule; math | https://aerawise.com/ |
| Gemini study notebooks / Gemini Notebook | Quizzes on uploaded class material, plan, progress dashboard; free year of AI Plus for eligible Canadian students until 31 Dec 2026 | Free tier; AI Plus after the free year | Spaced repetition, teach-back, verified quotes | https://blog.google/intl/en-ca/products/explore-get-answers/eligible-canadian-post-secondary-students-can-get-one-year-of-gemini-on-us/ |
| ChatGPT study mode | Socratic guidance, study guides, quizzes | In consumer plans [unverified] | Persistent schedule, teach-back grading | https://learn.chatgpt.com/docs/whats-new |
| Claude learning mode | Guides the student to think | In Claude plans [unverified] | Scheduling, grading against a course source | https://www.engadget.com/ai/anthropic-brings-claudes-learning-mode-to-regular-users-and-devs-170018471/ |
| RemNote | Exam scheduler, FSRS, AI flashcards, AI grading | Pro US$8/mo, Pro with AI US$18/mo (third party) | Teach-back | https://toolradar.com/tools/remnote/pricing |
| Knowt | AI voice tutor on files, flashcards | Conflicting listings [unverified] | Teach-back, verified quotes | https://toolradar.com/tools/knowt/pricing |
| Gizmo | Gamified AI flashcards, about 13 million learners, Series A Apr 2026 | About US$77/yr (third party) | Teach-back, rigor | https://doolpa.com/news/gizmo-22m-series-a-gamified-learning-april-2026 |
| Anki with FSRS | FSRS-6 default scheduler | Free desktop | Any AI checking | https://forums.ankiweb.net/t/updates-from-the-anki-team-june-2026/70120 |
| "Feynman AI" app cluster | Notes, quizzes, "explain it simply" | Mostly freemium | Source-verified grading, math | https://feynman-ai.com/ |
| Alice | AI tutor, study plans, graded mock exams | Free and Pro | Spaced repetition, teach-back | https://carrot.alice.tech/en-gb/pricing/ |

Takeaways: "teach the AI" is no longer a differentiator; the free baseline rose
for Canadian students; no page found claims verified citations for grading or
math-grade rigor (absence of evidence, not proof); Synapsi is the closest and
ships weekly.

Owed research: two hours of hands-on testing of Synapsi, AeraWise and Gemini
study notebooks on one of Dino's units; direct reading of Reddit threads.
