# P0-GATEPREP impl log

- Data: injections.jsonl (10), coverage.jsonl (20), leak_sessions.jsonl (30). No label file touched.
- evalcheck: suites.go (8 suites, budget), gate.go (gate.md and private file), eval.go (loaders: claim-level frozen labels, split, hashes), main.go (flags).
- Human labels count only if FROZEN.md records the labels_final.csv SHA-256.
- Spend cap: before each call, stop if spent + dearest call so far reaches the cap. First call has no estimate; llm.Meter.Cap is the hard backstop.
- Tests: `go test -race ./cmd/evalcheck/` ok. Full `go build ./... && go vet ./... && go test -race ./...` ok on a fresh migrated database.
- Dry run (fake checker and student, test split): 0 PASS, 2 FAIL, 1 PARTIAL, 5 NOT RUN. Correct: a fake that answers nothing passes nothing.
- Unverified: every real-model number; whether the real checker cites with source_id of the private unit; leak rule catches only 5-word copies.
- Needs a contract-free follow-up? No. Nothing in internal/, api/, migrations/ or web/ was changed.
