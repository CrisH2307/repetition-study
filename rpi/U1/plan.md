# U1 plan

1. Parser: scripts/parse_pdf.py, run once -> content/source.md, damaged-formulas.md. DONE
2. Concept list (8 concepts, 4-5 claims each, 3-5 required) in content/unit.json. DONE as draft. STOP: Cris approves list and demo concept.
3. content/allowlist.json = []. DONE
4. After approval: 20 explanations in eval/data/explanations.jsonl (10 correct, 5 partial, 5 wrong, shuffled, no labels in file).
5. After approval: labels_cris.csv, labels_dino.csv (empty label columns); scripts/check_labels.py.
6. After approval: content/seed.sql with demo history for the demo concept (next review about 7 days out).
Tests: `go test ./content/` (strict decode, ids, locators resolve, source text in sync, allowlist empty).
Risks: formulas damaged until Cris fixes source.md; claims paraphrase formulas in plain text; Q7 and Q8 share a concept.
