# U2 plan

1. internal/llm: retry helper + Gemini (Stream) + Anthropic (JSON), env config. Tests: retry unit, both clients against httptest servers.
2. internal/teach: StudentContext + prompt builder + probe selection + sentence cap. Tests: struct has no source fields, no source text in LLM requests.
3. Engine Start/Turn/Finish with Store interface + memory store. Async checker with timeout, Finish waits, absent fill, citation verify, ApplyTeachBack once.
4. leak_test.go (10 probes, fakes) and leak_real_test.go (tag realllm). TTFT log per turn.

Risks: real SDK calls unverified; no pg Store; checker results lag one turn; probe leak guard is a 5-word shingle check, not semantic.
