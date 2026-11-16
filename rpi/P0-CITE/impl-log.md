# P0-CITE impl log

- Regression test `TestFinishKeepsCitationAcrossLineBreak` added. Failed before the fix (citation nil at Finish).
- `internal/cite` added with table tests. `go test -race ./internal/cite` ok.
- Checker and `finalize` route through `cite.Verify`; `finalize` downgrades contradicted to partial. Old engine test quote (18 chars) lengthened to pass the F8 floor. Added `TestOneWordQuoteRejected`. `go test -race -count=1 ./internal/...` all ok.
- Review fix: finalize also drops a blank correction (and downgrades), matching the checker. Tests ok.
- Peer review: display blocks now need whitespace or text edges around delimiters; $x$$y$ and \\[2pt] tests added (failed before). Tests ok.
