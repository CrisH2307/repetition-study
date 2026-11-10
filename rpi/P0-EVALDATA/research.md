# P0-EVALDATA research

- Source: only `content/private/A3 MATH4430_6602.pdf` (practice set 3, 13 pages). Read with pdftotext into a scratch dir; nothing copied into tracked files.
- `content/unit.json` has 8 concepts; claims are mostly worked numeric answers (review 1.4 item 6, D20).
- Locator format is enforced by `content/unit_test.go` (`p.N, Ex 3.x` or `p.N, Qn`, optional ` (cont.)`).
- `content/seed.sql` uses claim ids 020101 to 020103, so those ids must stay.
- `TestEval` in `content/unit_test.go` expects 20 explanations and `labels_cris.csv`; it must move to claim-level CSVs.
- Gate bars (review 4.6): kappa >= 0.6 before freezing; >= 100 perturbed claims; tune only on a development split.
- The PDF never states the transient criterion for birth-death processes; left out of the claims.
- Python: stdlib only (csv, json, hashlib, unittest). No new dependency.
