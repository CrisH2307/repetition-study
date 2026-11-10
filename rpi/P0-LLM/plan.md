# Plan
1. internal/llm: both vendors do Stream and JSON; structured output; Gemini thinking set, cap 200.
2. models.json plus Meter (tokens, cost, latency, US$ cap), `llm.New` shared by server and evalcheck.
3. evalcheck: delete adapters, use `llm.New`, default unit `content/private/unit.json`, bounds-check loadLabels.
4. `realllm` tests (not run). Risks: structured-output and thinking settings unverified live.
Not done: server timeouts (this unit owns no server code); `llm_calls` table needs a contract change, ask Cris.
