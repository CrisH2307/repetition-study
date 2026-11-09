You own the tooling for P0-EVALDATA on branch feature/eval-data, from qa. Owners:
Dino and one domain classmate label; Claude builds tooling; Cris approves the
concept list. Files: eval/data/, content/private/ (gitignored), scripts/check_labels.py,
rpi/P0-EVALDATA/.

Read first: docs/STATUS.md (U1, C1), docs/DECISIONS.md, docs/ROADMAP.md (G0 table).
Decisions: D20 (claim rewrite), D10 (labellers), D9 (set sizes and kappa bar), D7
(where real-unit data lives), D24. Write plan.md, then STOP until Cris approves it.
Never write labels yourself; humans label.

Steps
1. [D20] With Dino, rewrite claims for at least 4 concepts as definitions,
   statements, conditions and methods (not numeric answers). Store the working
   pack under content/private/ if D7 says private; otherwise propose the diff to
   Cris (content/ is not in this boundary).
2. Hand-fix the damaged formula regions that rewritten claims cite (list in
   content/damaged-formulas.md), in the private copy.
3. Templates: eval/data/explanations.jsonl (id, concept_id, text), one label CSV
   per labeller at claim level (explanation_id, claim_id, label), a dev and test
   split file. If D7 makes the real unit private, real-unit files live in
   content/private/eval/ and only fixture-based files are committed.
4. scripts/check_labels.py: Cohen's kappa per label, disagreement list, freeze
   step that writes labels_final.csv and prints its SHA-256.
5. Perturbation generator spec (dropped condition, swapped parameter, wrong sign,
   wrong formula) and eval/data/leak_probes.jsonl (30 adversarial probes).

Tests
- check_labels.py on a toy pair of files gives a known kappa.
- Shapes match what cmd/evalcheck reads (check its loaders).

Done when: [D9] kappa at least the decided bar, files frozen, hash recorded by
Cris in docs/DECISIONS.md.

Report in under 200 words: done, tests with results, risks, unverified.
