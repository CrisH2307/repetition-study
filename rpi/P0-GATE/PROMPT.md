Unit P0-GATE, wave 3. Branch `feature/gate-results` from qa. Model: Sonnet 5.5, effort medium. Cris runs this and Dino reads it.
Files: `eval/results/`, `rpi/P0-GATE/`.
Needs: real keys in `.env`, Dino's frozen `labels_final.csv`, P0-LLM and P0-INT merged. If any is missing, stop and say which.
Read first: `docs/PLAN.md` gate table, `docs/improved/strategy-review-fable.md` section 4.6, `rpi/P0-EVALDATA/PROMPT.md`.

Steps
1. Run `cmd/evalcheck` for each candidate model: Claude Sonnet 5.5, Claude Haiku 5.5, and a Gemini Pro-class and Flash-class model (verify the ids in the vendor model list first). Run each suite in the PLAN gate table that has data: human set, perturbation set, citation quality, stability (3 runs), injection, coverage gradient, leak (30 sessions), cost and latency.
2. Spend cap for the whole bake-off: US$50. Stop and tell Cris before you pass it.
3. Write `eval/results/gate.md`. Start with a plain-language summary for Dino (pass or fail per row, what it means, no jargon). Then the numbers per model with the commit hash and the dataset hash, cost per session, latency, and every failed example in full.
4. Do not tune prompts, change code or edit labels to pass. Tuning is allowed only on the development split, and only if Cris asks after reading the report.
5. Stop and ask Cris what to do. Never mark a bar as passed without a measurement.

Push the branch, then open a PR into qa with `gh pr create --base qa` (body: "Closes #<issue>", tests run, what is unverified). Never merge. Stop. Report under 200 words.
