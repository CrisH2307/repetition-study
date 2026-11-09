You own P0-HYG on branch chore/repo-hygiene, from qa. Owner: Cris; Claude drafts.
Files: .github/, .gitignore, CLAUDE.md, docs/, .claude/, content/, scripts/unit.sh,
rpi/P0-HYG/. Do not touch other paths.

Read first: docs/STATUS.md, docs/DECISIONS.md, docs/ROADMAP.md (Phase 0).
Decisions: D7 (course material), D8 (regulations), D24 (Phase 0 plan). Steps marked
[D7] wait until D7 is Decided. Write plan.md, then STOP until Cris approves it
(regulation 5).

Already done on chore/roadmap-v2: docs/STATUS.md, DECISIONS.md, ROADMAP.md,
STRATEGY.md, CLAUDE.md "Current phase". Do not redo them.

Steps
1. .github/workflows/ci.yml: go build, go vet, go test with a Postgres 16 service
   and DATABASE_URL, go test -race. Verify current action versions from their docs.
2. .github/pull_request_template.md: unit name, one concern, tests run, decisions
   touched (D-ids).
3. Write the exact branch protection settings for main and qa (PR required, CI
   required, no direct push) into rpi/P0-HYG/plan.md. Cris applies them in GitHub;
   you never change repo settings.
4. docs/TESTING.md: B4 must not match itself (real key shapes such as
   `AIza[0-9A-Za-z_-]{30,}`, exclude docs/); align the gate report file name with
   what P0-GATE writes (eval/results/gate.md).
5. .claude/skills/status/SKILL.md: replace the 5pm deadline with "Phase 0 dates in
   docs/ROADMAP.md". Update argument hints in .claude/skills/unit/SKILL.md and
   scripts/unit.sh to list P0-* units.
6. [D7] Move content/source.md and the source text in content/unit.json to the
   gitignored content/private/. Add a licensed or team-written fixture so
   `go test ./content/` still passes. Record the fixture's licence. History purge
   is a separate manual step: write the commands for Cris, do not run them.

Tests
- CI passes on the PR. `git grep` for a distinctive sentence of the practice set
  returns nothing on the branch [D7]. The new B4 command prints nothing on qa.

Done when: CI green on the PR, Cris confirms protection is on (a direct push to qa
is rejected), D7 steps done or explicitly deferred in impl-log.md.

Report in under 200 words: done, tests with results, risks, unverified.
