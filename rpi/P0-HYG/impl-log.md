# P0-HYG impl log

- Added `.github/workflows/ci.yml` (build, vet, `make migrate`, `go test -race ./...`, Postgres 16 service, no secrets). YAML parses.
- Moved `source.md`, `seed_pack.sql`, `damaged-formulas.md` and full `unit.json` to gitignored `content/private/` (confirmed with `git check-ignore`). Tracked `unit.json` keeps concepts and claims (our wording) with the source text empty and a neutral title.
- Added `content/fixture/{notes.md,unit.json}` (original text) and `content/README.md`. `unit_test.go` runs the fixture always, private checks skip when absent.
- `go test -race ./...` passes with and without `content/private/`.
- Left alone: `content/seed.sql`, `eval/data/explanations.jsonl` (our own wording, no course text found).

## History purge steps for Cris (not run)
Course text is still in git history. On a FRESH clone (never your working copy):
```
git clone git@github.com:CrisH2307/repetition-study.git purge && cd purge
git filter-repo --invert-paths --path content/source.md --path content/seed_pack.sql --path content/damaged-formulas.md --path-glob 'content/*.pdf'
```
`content/unit.json` and `scripts/` history also hold course text in older versions: to scrub it, add `--path content/unit.json` and re-add the cleaned file in a new commit afterwards.
Then force-push all branches and tags (`git remote add origin ...; git push --force --all; git push --force --tags`), tell Dino and every agent to re-clone, and delete old PR refs on GitHub (or contact support to purge cached views). Do this when no PR is open.
