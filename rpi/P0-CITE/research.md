# P0-CITE research

- F1 confirmed: `checker.verifyCitation` compared whitespace-normalized text, `teach.finalize` used raw `strings.Contains`. A quote across a line break passed the checker and was dropped at Finish, leaving "contradicted" with no quote.
- `domain.Citation` has SourceID, Quote, Locator. No new field needed.
- `domain.SourceBlock.TextNorm` comment already names "the one normalizer in internal/cite".
- Sources are text/Markdown. Display formulas are `$$...$$` or `\[...\]` (assumed, unverified against the real parsed unit, which is gitignored).
- Normalize stays whitespace-only (no case folding, no quote-mark mapping), as the checker tests already require.
