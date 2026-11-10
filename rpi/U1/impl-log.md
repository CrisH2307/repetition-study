# U1 impl-log
- Parser written, run once with pdftotext fallback: 13 pages -> content/source.md.
- unit.json draft: 8 concepts, 35 claims (32 required). `go test ./content/` ok.
- allowlist.json = [] ; test pins its fields.
- Explanations: 20 (10 correct, 5 partial, 5 wrong intended), shuffled, ids e01..e20, no labels in the file. Answer key kept OUTSIDE the repo in the shared notes folder so the humans label blind.
- Label sheets + scripts/check_labels.py: tested on a temp copy with fake labels (agreement, disagreements, --init-final, completeness). Real labels not yet filled.
- seed_pack.sql + seed.sql: ran migrations and both seeds twice on postgres:16; 8 concepts, 35 claims, 3 review events, next review 7 days out. FSRS values computed with go-fsrs v4.0.0 defaults, short-term off.
- `go test ./content/` ok.
