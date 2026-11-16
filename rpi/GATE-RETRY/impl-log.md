# Gate retry prep (#65, #66)

- #65: `untrusted()` in internal/checker now turns the `<` of any fence-like
  tag (`transcript`, `taught`, `latex`, any case, spaces, slashes or
  backslashes, open or close, with or without `>`) into `&lt;`. Prompt rule
  text unchanged. Table test covers each variant and the i03 attack.
- #66: OpenRouter JSON calls (only the checker uses them) send
  `temperature: 0` and `seed: 7`. Student streams unchanged.
  Docs read 2026-11-15: openrouter.ai/docs/api-reference/parameters.
  Seed determinism "is not guaranteed for some models".
- Unverified: the GPT 6 Luna model page lists `seed` but not `temperature`.
  Default routing should ignore unknown parameters, but the same page says a
  400 comes back for unsupported ones. If the first gate item fails with
  status 400 naming temperature, stop and tell Cris (drop that one line).
- Tests: go build, go vet, go test -race on a fresh database: all pass.

Next runs (Cris):
    go run ./cmd/evalcheck --models "$CHECKER_MODEL" --student-model "$STUDENT_MODEL" --split dev --max-cost 2
    go run ./cmd/evalcheck --models "$CHECKER_MODEL" --student-model "$STUDENT_MODEL" --max-cost 2
