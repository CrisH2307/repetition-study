# P0-OPENROUTER (issue #57)

Done: `internal/llm/openrouter.go` (chat completions, streaming and JSON, usage from the reply, cut-short streams metered at the cap), routing by the `openrouter/` id prefix, two priced entries in `models.json`, httptest tests, D21, GATE.md env names.

Docs checked 2026-11-14 (openrouter.ai/docs): endpoint `POST /api/v1/chat/completions`, `Authorization: Bearer`, SSE with `: OPENROUTER PROCESSING` keep-alives, usage in the last chunk before `[DONE]`, `response_format` json_schema.

Unverified (no live call made): `reasoning.enabled=false` accepted by the Gemini route; strict=false json_schema accepted by the Claude route; whether `completion_tokens` includes reasoning tokens (docs imply billed as output); `.env.example` has no OPENROUTER_API_KEY line (outside this unit's boundary).
