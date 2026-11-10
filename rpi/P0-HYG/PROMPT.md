Unit P0-HYG, wave 0. Branch `chore/repo-hygiene` from qa. Model: Sonnet 5.5, effort low.
Files: `.github/`, `.gitignore`, `CLAUDE.md`, `docs/`, `.claude/`, `content/`, `rpi/P0-HYG/`.
Read first: `docs/PLAN.md`, `docs/DECISIONS.md` (D7, D8), `docs/improved/strategy-review-fable.md` section 3.6 and finding 5.

Why: a professor's practice set is committed in `content/source.md` and `content/unit.json`, which breaks our own rule and D7. CI does not exist.

Steps
1. Add `.github/workflows/ci.yml`: Go build, vet, `go test -race ./...` with a Postgres 16 service and `make migrate` first. No deploy, no secrets.
2. Remove course text from tracked files: `content/source.md`, `content/seed_pack.sql` (it embeds the source text), the `text` fields inside `content/unit.json`, and `content/damaged-formulas.md`. Check `content/seed.sql` and `eval/data/explanations.jsonl` too and leave only text that is our own wording. Keep the concept and claim structure. Move the real files to gitignored `content/private/` on this machine (do not delete the only copy: the PDF stays in `content/private/`).
3. Write a short original fixture, `content/fixture/notes.md` (about one page on exponential races and Poisson superposition, in your own words, with 6 or more distinct sentences a quote can match) and a `content/fixture/unit.json` that cites it. Make `content/unit_test.go` use the fixture so CI passes without private files. Document in `content/README.md` where the private copy goes and that `make seed` reads it.
4. Write the history purge steps for Cris into `rpi/P0-HYG/impl-log.md` (use `git filter-repo` on a fresh clone, list the paths). Do not run it. Cris runs it by hand.
5. Sweep the docs for broken references (old deadlines, `rpi/U7`, `rpi/RELEASE`, `rpi/INT`, `rpi/GATE` paths, the old gate file name) and fix them. Do not rewrite `docs/PLAN.md`, `DECISIONS.md` or `TESTING.md`; list anything wrong in them in your report instead.

Done when: `go test ./...` passes with no private files present; `git grep -n "Practice Problem Set\|local board game store" -- . ":!docs/improved"` prints nothing; CI file is valid YAML.
Do not open a PR. Push the branch and stop. Report under 200 words.
