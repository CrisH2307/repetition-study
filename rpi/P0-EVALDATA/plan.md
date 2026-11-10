# P0-EVALDATA plan

1. Propose 4 concepts x 4 claims (`claims-proposal.md`). STOP for Cris.
2. Update those 4 concepts in `content/unit.json`; keep ids.
3. `eval/data/explanations.jsonl`: 20 explanations (10 correct, 5 partial, 5 wrong), shuffled, no labels.
4. `labels_dino.csv`, `labels_classmate.csv`: columns id, claim_id, label, note; one row per (explanation, claim). Delete `labels_cris.csv`.
5. `scripts/check_labels.py`: kappa overall and per label, disagreements, `--freeze` (refuse below 0.6, write `labels_final.csv`, SHA-256 in `FROZEN.md`). Test: `scripts/test_check_labels.py` with fake files.
6. `eval/data/perturbations.jsonl` (>= 100, tagged error type) and `eval/data/split.json` (dev/test).
7. `eval/data/LABELLING.md` for Dino.
8. Update `TestEval` in `content/unit_test.go`.

Risks: claim wording needs a domain check by Dino; the answer key for explanations is not committed (would leak labels), so the checker eval relies on Dino's labels.
