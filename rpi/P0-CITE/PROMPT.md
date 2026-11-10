Unit P0-CITE, wave 1. Branch `fix/citation-normalize` from integration/p0 (or qa). Model: Opus 5.5, effort high. Run `/code-review medium` on your branch before you report.
Files: `internal/cite/`, `internal/checker/`, `internal/teach/engine.go`, `internal/teach/engine_test.go`, `rpi/P0-CITE/`.
Read first: `docs/improved/strategy-review-fable.md` section 1.4 items 2 and 3 and section 4.2 (F1, F8), `internal/checker/verify.go`, `internal/teach/engine.go` near `finalize`.

Bug F1: the checker verifies a quote after whitespace normalisation, then `finalize` re-checks with raw `strings.Contains`. Sources have line breaks, so a verified citation is dropped at Finish and the report shows "contradicted" with no quote.

Steps
1. Write a failing regression test first: a multi-line source, a quote that spans the break, run through Check and Finish. It must keep the citation.
2. Create `internal/cite` with exactly one `Normalize` and one `Verify(sourceText, quote)`. The checker and the teach engine both call it. Delete every other quote check.
3. In `finalize`, when a citation fails verification, downgrade `contradicted` to `partial`, as the checker already does.
4. Fix F8: reject a quote shorter than 20 characters after normalisation, unless it is a whole display-formula block. Add tests for a one-word quote and for garbled-math quotes.

Do not change the contract. If `Citation` needs a new field, stop and ask Cris.
Done when: new tests fail before and pass after; `go test -race ./internal/...` is green; `git grep -n "strings.Contains" internal/teach internal/checker` shows no quote check outside `internal/cite`.
Do not open a PR. Push the branch and stop. Report under 200 words.
