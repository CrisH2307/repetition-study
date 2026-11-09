You own P0-CITE on branch fix/citation-normalize, from qa. Owner: Claude; Cris
reviews. Files: internal/cite/, internal/checker/, internal/teach/engine.go,
internal/teach/engine_test.go, rpi/P0-CITE/.

Read first: docs/STATUS.md (F1, F8), docs/DECISIONS.md, docs/ROADMAP.md.
Decisions: D22 (minimum quote length) gates step 4 only; D24. Write plan.md, then
STOP until Cris approves it (regulation 5).

Bug F1: internal/checker/verify.go:35-44 verifies quotes after whitespace
normalization; internal/teach/engine.go:299 re-checks with raw strings.Contains,
so a quote spanning a line break is dropped at Finish and the label stays
contradicted with no quote.

Steps
1. New package internal/cite with one Normalize and one Verify(quote, sourceID,
   approved map) used by both checker and teach. No other normalization anywhere.
2. Checker uses cite.Verify. Behaviour unchanged otherwise.
3. teach.finalize uses cite.Verify and, when it drops a citation, downgrades
   contradicted to partial, same rule as the checker (verify.go enforce).
4. [D22] Reject quotes shorter than the decided minimum after normalization.
   Use a named constant. Skip this step if D22 is still Pending and say so.

Tests
- Regression: source "alpha beta\ngamma delta", quote "beta gamma": citation
  survives Finish, label unchanged.
- Dropped citation at Finish turns contradicted into partial.
- [D22] One-word quote rejected.
- go test -race ./internal/... green.

Done when: all tests above pass and there is exactly one citation normalization
function in the repo (grep proves it; internal/teach/student.go oneLine is a
display helper, not citation matching, and stays).

Report in under 200 words: done, tests with results, risks, unverified.
