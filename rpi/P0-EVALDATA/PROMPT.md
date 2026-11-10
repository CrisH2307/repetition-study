Unit P0-EVALDATA, wave 1, after P0-HYG is merged into qa. Branch `feature/eval-data`. Model: Opus 5.5, effort medium for the math content, then Sonnet 5.5 for the tooling.
Files: `eval/data/`, `content/`, `scripts/check_labels.py`, `rpi/P0-EVALDATA/`.
Read first: `docs/improved/strategy-review-fable.md` sections 1.4 items 6 to 8, 4.6 and D10, D20; `docs/PLAN.md` gate table; `content/unit.json`. The real PDF is in gitignored `content/private/`.

Already on qa from U1: `eval/data/explanations.jsonl` (20 explanations, no claim-level split), `labels_cris.csv`, `labels_dino.csv` (empty, explanation level) and `scripts/check_labels.py`. D10 says Cris does not label, and labels are per claim, so replace `labels_cris.csv` with `labels_classmate.csv` and extend the script. Reuse the explanations only if they still fit the rewritten claims.

Rules: never commit text copied from the PDF. Student-style explanations are written in your own words. Labels are claim level: for each explanation, each required claim gets `supported`, `partial`, `contradicted` or `absent`. Dino and one classmate label independently. Cris does not label. Dino is not technical: anything for him is plain language with exact steps.

Steps
1. Rewrite claims for 4 concepts as definitions, statements, conditions and methods (D20), not worked numeric answers. Keep each claim anchored to a private-source locator. Show the 4 concepts and claims to Cris and STOP for approval before step 2.
2. After approval, draft 20 explanations (about 10 correct, 5 partly correct, 5 wrong; about 80 claim labels), shuffled, with no labels inside the file: `eval/data/explanations.jsonl`.
3. Create `eval/data/labels_dino.csv` and `labels_classmate.csv` with empty label columns.
4. `scripts/check_labels.py`: Cohen's kappa overall and per label, a list of disagreements, and a `--freeze` option that writes `labels_final.csv` and records its SHA-256 in `eval/data/FROZEN.md`. Refuse to freeze below kappa 0.6.
5. `eval/data/perturbations.jsonl`: at least 100 claims auto-changed (dropped condition, swapped parameter, wrong sign, wrong formula), each tagged with its error type. Add a development and test split file. Tuning is allowed only on the development split.
6. A one-page plain-language guide for Dino, `eval/data/LABELLING.md`, with an example of each label.

Done when: the files exist, `check_labels.py` has a test with fake label files, and Dino has the guide. The real labels are Dino's task, not yours.
Push the branch, then open a PR into qa with `gh pr create --base qa` (body: "Closes #<issue>", tests run, what is unverified). Never merge. Stop. Report under 200 words.
