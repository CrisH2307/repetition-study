# P0-GATEPREP impl log

- Data: injections.jsonl (10), coverage.jsonl (20), leak_sessions.jsonl (30). No label file touched.
- evalcheck: suites.go (8 suites, budget), gate.go (gate.md and private file), eval.go (loaders: claim-level frozen labels, split, hashes), main.go (flags).
- Human labels count only if FROZEN.md records the labels_final.csv SHA-256.
- Spend cap: before each call, stop if spent + dearest call so far reaches the cap. First call has no estimate; llm.Meter.Cap is the hard backstop.
- Tests: `go test -race ./cmd/evalcheck/` ok. Full `go build ./... && go vet ./... && go test -race ./...` ok on a fresh migrated database.
- Dry run (fake checker and student, test split): 0 PASS, 2 FAIL, 1 PARTIAL, 5 NOT RUN. Correct: a fake that answers nothing passes nothing.
- Unverified: every real-model number; whether the real checker cites with source_id of the private unit; leak rule catches only 5-word copies.
- Needs a contract-free follow-up? No. Nothing in internal/, api/, migrations/ or web/ was changed.

## Review fixes (issue #52, branch fix/gate-prep-review)
- Cap: every client is wrapped (guard.go). A call starts only if spend + worst case (bytes as tokens, full output cap, all retries) stays under the cap. Unpriced real models are refused while a cap is set.
- internal/llm: Gemini and Anthropic meter a stream cut short at the reply cap (Call.CutShort). Cost row runs one full session per concept (8 turns + Finish), bar on the dearest.
- Perturbations: 146 total, 114 test (37 new from explanation sentences). Row is NOT RUN below 100.
- gate.md: no checker student_text, no AI student replies. Private file only.
- FROZEN.md: only `<data>/FROZEN.md`, exact `- SHA-256:` line.
- Verdicts only on `--split test`; other splits are DEV RUN.
- Unverified: worst-case token rule (one token per byte) and the +1000 overhead; whether Gemini thinking sits inside MaxOutputTokens; real billing of a canceled stream (metered at the cap, an upper bound).
