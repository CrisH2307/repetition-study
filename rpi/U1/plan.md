# U1 plan

1. Parser: scripts/parse_pdf.py, run once -> content/source.md, content/damaged-formulas.md. DONE
2. Concept list: 8 concepts, 35 claims in content/unit.json. DONE. Dino said ok (relayed by Cris, issue 7 has no comment).
3. content/allowlist.json = []. DONE
4. eval/data/explanations.jsonl: 20 explanations, shuffled, no labels in the file. DONE
5. labels_cris.csv, labels_dino.csv (id,label,note, empty labels) and scripts/check_labels.py. DONE. labels_final.csv is made by `check_labels.py --init-final` once both fill their sheets.
6. content/seed_pack.sql (generated: subject, source, concepts, claims) and content/seed.sql (cards, schedule, 3 review events for the demo concept, next review 7 days out). DONE
Tests: `go test ./content/` (strict decode, ids, locators resolve, source in sync, seed_pack fresh, eval files line up). Seeds also run twice on a throwaway Postgres 16 with migrations 0001 and 0002.
Open: Cris and Dino fill labels (blind) and settle differences; formulas in source.md still damaged.
