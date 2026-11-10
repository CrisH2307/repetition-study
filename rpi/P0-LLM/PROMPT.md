Unit P0-LLM, wave 1. Branch `feature/llm-parity` from integration/p0 (or qa). Model: Sonnet 5.5, effort medium.
Files: `internal/llm/`, `cmd/evalcheck/`, `.env.example`, `rpi/P0-LLM/`.
Read first: `docs/improved/strategy-review-fable.md` section 4.2 (F9, F10, F14), 4.5 and 4.6, `internal/llm/`, `cmd/evalcheck/llm.go`.

Problem: `cmd/evalcheck` has its own model adapters and prompt suffix, so passing the eval would not certify the shipped checker. Gemini and Anthropic are each wired for only one role.

Steps
1. `cmd/evalcheck` must import `internal/llm` and `internal/checker` and call the same code the server calls. Delete its private adapters.
2. Make `internal/llm` provider neutral: either vendor can fill any role (`Stream` and `JSON` on both). Verify the current SDK calls in the module cache before using them, and mark anything you could not run as unverified.
3. Gemini: set the thinking configuration explicitly and raise the token cap to about 200. Verify on the real API only if `GEMINI_API_KEY` is set. If it is not, mark this unverified.
4. Use structured output for JSON replies instead of scraping JSON from text. Bounds-check `loadLabels`. Add server timeouts where this unit owns code.
5. Model ids and prices live in a config file (`internal/llm/models.json`), never in code. Meter every call (input tokens, output tokens, model, cost) in memory and print cost per session from `evalcheck`. A database `llm_calls` table needs a contract change: ask Cris.
6. Real-model tests sit behind the build tag `realllm` so CI never spends money. Hard cap: stop a run at US$5 for this unit.

Done when: `evalcheck --model fake` runs through `internal/llm`; if keys exist, one real model per vendor runs and the cost per session prints; `go test ./internal/llm/... ./cmd/evalcheck/...` is green.
Do not open a PR. Push the branch and stop. Report under 200 words, listing what is unverified for lack of keys.
