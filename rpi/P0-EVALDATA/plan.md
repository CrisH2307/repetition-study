# P0-EVALDATA plan

1. Propose 4 concepts x 4 claims (`claims-proposal.md`). Approved by Cris.
2. Update those 4 concepts in `content/unit.json`; keep ids.
3. `eval/data/explanations.jsonl`: 20 explanations (10 correct, 5 partial, 5 wrong), shuffled, no labels.
4. `labels_dino.csv`, `labels_classmate.csv`: columns id, claim_id, label, note; one row per (explanation, claim). Delete `labels_cris.csv`.
5. `scripts/check_labels.py`: kappa overall and per label, disagreements, `--freeze` (refuse below 0.6, write `labels_final.csv`, SHA-256 in `FROZEN.md`). Test: `check_labels.py --selftest` with fake files (keeps the test inside the boundary).
6. `eval/data/make_perturbations.py` builds `perturbations.jsonl` (>= 100, tagged) and `split.json` (dev/test).
7. `eval/data/LABELLING.md` for Dino.
8. Update `TestEval` in `content/unit_test.go`.

Risks: claim wording needs a domain check by Dino; the answer key for explanations is not committed (would leak labels), so the checker eval relies on Dino's labels.
