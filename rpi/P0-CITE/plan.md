# P0-CITE plan

1. Failing regression test in `internal/teach/engine_test.go`: real `checker.Checker` on a fake LLM, multi-line source, quote across the break, Check then Finish keeps it.
2. `internal/cite`: `Normalize`, `Verify(sourceText, quote)`, `MinQuote = 20`, whole display formula exempt. Table tests incl. one-word and garbled math.
3. Checker `verifyCitation` and teach `finalize` call `cite.Verify`. Delete the checker's own `normalize`.
4. `finalize` downgrades contradicted to partial when no citation survives.

Risks: the 20-char floor may reject real short but valid quotes; display-formula detection is regex on raw source text, not on `source_blocks` (no block ids in `Citation` yet, see M-CHECK2).
