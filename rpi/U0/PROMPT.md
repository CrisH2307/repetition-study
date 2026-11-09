You are setting up the repo CrisH2307/repetition-study on branch feature/setup, created from qa.
If qa does not exist, create it from main. Do not commit to main.

Create:
1. Keep CLAUDE.md, .claude/, docs/, scripts/ and rpi/*/PROMPT.md from the starter
   kit. Do not rewrite them.
2. compose.yaml with Postgres 16, .env.example with variable names only, a
   .gitignore covering .env, and a Makefile with: up, migrate, test, run, seed.
3. migrations/ with: subjects, concepts, claims, sources, teach_sessions,
   teach_turns, taught_claims, claim_labels, cards, concept_schedules,
   review_events, plan_blocks, digests. Add a database trigger that rejects
   UPDATE and DELETE on review_events.
4. internal/domain/ with the shared types and interfaces below.
5. api/openapi.yaml for the endpoints below, plus cmd/api with /healthz.
6. Fake implementations of every interface in internal/fakes, so units can
   build and test alone. The fake LLM must be deterministic.
7. Add go-fsrs, pgx and the official Gemini and Anthropic Go SDKs to go.mod now,
   and verify their real module paths, so later units never touch go.mod.
8. Create an empty research.md, plan.md and impl-log.md beside each PROMPT.md.

A single seeded demo user is fine. There is no auth today.

Domain types
- ClaimLabel: supported, partial, contradicted, absent
- Grade: again, hard, good, easy
- EventType: card_review, teach_back, manual_override
- SourceKind: course, allowlist
- Source: id, kind, title, url, publisher, license, reviewed_by, reviewed_at, text
- Subject: id, name, weekly_hours, level (beginner|intermediate|advanced),
  exam_date, weight
- Concept: id, subject_id, name, claims
- Claim: id, concept_id, text, source_id, locator, required
- TaughtClaim: id, session_id, concept_id, student_text, turn_no
- Citation: source_id, quote, locator
- ClaimResult: claim_id (nullable), student_text, label, citation (nullable),
  correction (nullable), probe_hint (nullable, never contains source text)
- ReviewEvent: id, event_type, concept_id, card_id (nullable), grade (nullable),
  due_before, due_after, stability_before, stability_after, payload, created_at
- PlanBlock: id, subject_id, concept_id (nullable), kind (review|teach|new),
  starts_at, minutes, reason, locked

Interfaces, all taking context.Context
- LLM: Stream(req) returns a channel of deltas; JSON(req, out) fills a struct
- TeachEngine: Start(concept_id); Turn(session_id, transcript, latex) streams
  deltas; Finish(session_id) returns Report
- Checker: Check(CheckInput) returns []ClaimResult
- Scheduler: ApplyTeachBack(concept_id, []ClaimResult); ReviewCard(card_id,
  grade); Override(concept_id, due_at, reason); Due(now)
- Planner: Generate(user_id, week_start) returns Plan; FurtherPlanning(week_start)
- Digester: Today(now)
- Clock: Now(), so all date logic is testable

Endpoints under /v1, JSON unless noted
- POST and GET subjects
- POST plan/generate; GET plan?week=; PATCH plan/blocks/{id}
- POST teach/sessions; POST teach/sessions/{id}/turns returns text/event-stream
  with events token, done, error; POST teach/sessions/{id}/finish;
  GET teach/sessions/{id}/report
- GET reviews/due; POST cards/{id}/review; PATCH cards/{id}
- POST concepts/{id}/override
- GET analytics/further-planning?week=
- GET digest/today

Done when `make up migrate test` passes on a clean checkout and the trigger is
proven by a test. Open a small PR series into qa. Report in under 200 words and
list anything in this contract you think is wrong.
