# U1 research

- Input: `A3 MATH4430_6602.pdf`, 13 pages, TeX (pdfTeX 1.40.25), the course practice set 3 with worked solutions. Copied to content/private/ (gitignored).
- Mismatch with docs/PLAN.md: this is NOT Chapter 2 discrete-time Markov chains. It is Lawler ch. 3 (Ex 3.2, 3.3, 3.5, 3.8, 3.11, 3.14) plus extra Q1 to Q8 on Poisson processes, exponential races, CTMC generators, balance equations and birth-death processes. Concepts come from this PDF only. Cris to confirm.
- Docling: PyPI 2.149.0, license MIT (METADATA says `License-Expression: MIT`). Not installed. It depends on torch and model downloads, so heavy for a one-off. Its API (`DocumentConverter().convert(path).document`, `export_to_markdown`) is UNVERIFIED here: never imported or run.
- Fallback used: `pdftotext -layout` (poppler, already installed). `scripts/parse_pdf.py` tries Docling first, falls back to pdftotext. Run once, output in content/source.md with `<!-- page N -->` markers.
- Math is damaged (fractions, sums, matrices, exponents split across lines). Full list in content/damaged-formulas.md. Cris fixes by hand, then `--sync`.
- Proofs cite "lecture note 3" (Proposition 1.1, Examples 2.1 and 2.4), which we do not have. Claims state only what the PDF states and name those citations as the PDF names them.
- I recomputed each worked answer used in a claim (Ex 3.8, Q1, Q3, Q4, Q5, Q7, Q8, Ex 3.11 sums). All match the PDF. Q2 has a typo in the PDF (see damaged-formulas.md) so it is left out.
- unit.json shape is mine (domain has no Unit type): `{subject, sources[], concepts[] with nested claims}`. Decoded strictly by content/unit_test.go.
- Source text in unit.json is source.md with spaces collapsed and blank lines dropped, so a quote is not broken by layout padding.
- Allowlist is `[]`. Fields per entry (url, publisher, license, reviewed_by, reviewed_at) are enforced by the test.
- Unverified: whether Docling gives better math than pdftotext on this file.
- Eval meaning (from cmd/evalcheck): one label per explanation = worst result over the concept's required claims. So a "correct" explanation covers every required claim of its concept.
- Seed history uses go-fsrs v4.0.0 default weights, short-term off (same as internal/srs). Gaps 1, 3 and 8 days. The sheet "Study - F26" tab MATH 6602 shows Cris's rhythm 1, 2, 3, 4, 5, 7, 15, 30, 45 days. The sheet also has a continuous-time Markov chain column.
- Loading: `make seed` only loads scripts/seed.sql. content/seed_pack.sql then content/seed.sql must be loaded by INT (not in U1's boundary).
