You own P0-LLM on branch feature/llm-parity, from qa. Owner: Claude, run with
Cris's keys on Cris's machine. Files: internal/llm/, cmd/evalcheck/, .env.example,
rpi/P0-LLM/.

Read first: docs/STATUS.md (F2, F9, F10, F14), docs/DECISIONS.md, docs/ROADMAP.md.
Decisions: D11 (models and US$50 bake-off cap) gates real calls; D24. Write
plan.md, then STOP until Cris approves it (regulation 5). Keys come from the
environment only; never print or commit them.

Steps
1. internal/llm: JSON and Stream on both vendors (Gemini JSON with a response
   schema or MIME type; Anthropic Stream), so either vendor can fill any role.
   Verify every SDK call against the module version in go.mod.
2. Gemini: set thinking configuration explicitly and make MaxTokens configurable
   (F10). Verify behaviour on the real API and record it.
3. Robust JSON extraction: first complete JSON value, not just fence stripping
   (internal/llm/anthropic.go:90).
4. Token metering on every call (input, output, cached) through a Meter type in
   internal/llm that the caller reads. Do not change the domain.LLM interface;
   internal/domain belongs to P0-CONTRACT.
5. cmd/evalcheck imports internal/llm and internal/checker; delete its private
   adapters (cmd/evalcheck/llm.go). Same prompt as production.
6. evalcheck prints cost per session (prices from flags or env, never hardcoded)
   and fixes the short-row panic (cmd/evalcheck/eval.go:87).
7. evalcheck leak mode: runs the real student through internal/teach exported
   types on probes from eval/data/leak_probes.jsonl (written by P0-EVALDATA) and
   reports any reply that states an untaught fact. Read-only use of internal/teach.
8. .env.example: add STUDENT_MODEL, CHECKER_MODEL, MATCHER_MODEL names only.

Tests
- httptest stubs for every new method; `evalcheck --model fake` on temp data.
- [D11] One real call per vendor through internal/llm; log model id, latency,
  tokens, cost. Stop at the budget cap.

Done when: fake and one real model per vendor run through the production path, cost
per session printed, results recorded in impl-log.md.

Report in under 200 words: done, tests with results, risks, unverified.
