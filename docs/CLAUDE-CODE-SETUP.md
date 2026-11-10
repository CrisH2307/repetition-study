# Running agents

## Starting a unit

One chat per unit, each in its own worktree so files never collide.

```bash
scripts/unit.sh P0-CITE
```

`scripts/unit.sh` makes a worktree from `qa` on the unit's branch and starts Claude Code with `/unit <name>`. For units that need earlier wave work, start from the staging branch instead:

```bash
git switch integration/p0 && scripts/unit.sh P0-OUTAGE
```

## Who merges

Agents push their branch and stop. They never open PRs and never touch `qa` or `main`. The main agent merges reviewed branches into `integration/p0`. Cris merges `integration/p0` into `qa`, tests, then `qa` into `main`.

## During the build

| Need | Do |
| --- | --- |
| Context past 50% | `/handoff <unit>`, open a new chat, then `/unit <unit>` |
| Change a task | Edit `rpi/<unit>/PROMPT.md`, not the chat |
| New rule for every chat | Add one line to `CLAUDE.md` |
| New unit | Add `rpi/<name>/PROMPT.md` and a `.claude/boundaries.json` entry |
| Contract change | Stop and ask Cris. Only P0-CONTRACT edits `internal/domain`, `api/` or `migrations/` |

## Which model for which task

Prices per million tokens, input and output, from the Claude API skill cache of 2026-11-03: Opus 5.5 $4 and $20, Sonnet 5.5 $2 and $10, Haiku 5.5 $0.10 and $0.50, Fable 5.2 $8 and $40. Check the live pricing page before relying on these. In Claude Code pick the model in the model menu and effort with the effort control. Default to the cheaper model, and move up only for the reason in the right column.

| Task | Model | Effort | Why |
| --- | --- | --- | --- |
| Main agent (plans, reviews, merges, reports) | Opus 5.5 | high | Little text, but a mistake spreads to every unit |
| P0-CONTRACT | Opus 5.5 | high | Everything depends on it |
| P0-CITE, P0-OUTAGE | Opus 5.5 | high | Small, subtle, the product's trust rests on it |
| P0-EVALDATA (writing the explanations and error cases) | Opus 5.5 | medium | Needs correct graduate math. Dino checks it |
| P0-LLM, P0-STORE, P0-SRS, P0-TZ, P0-INT, P0-SLICE, P0-GATE | Sonnet 5.5 | medium, high for P0-INT | Well specified work. Move P0-INT to Opus 5.5 if it fails twice |
| P0-HYG, docs, plain-language text | Sonnet 5.5 | low | Mostly writing and config |
| Running tests, grepping, summarising output | Haiku 5.5 | low | Cheap and mechanical |
| Code review of CITE and OUTAGE diffs | Opus 5.5 | medium | `/code-review medium` on the branch |

Fable 5.2 costs more than Opus 5.5. Use it only for one-off architecture reviews, not for building.

Ways to spend fewer tokens: give each agent only its own prompt and the files it names, not the whole review; tests use the fake model, so real vendor calls happen only in P0-LLM and P0-GATE under the US$50 cap; stop and hand over at 50% context; re-read `rpi/<unit>/impl-log.md` instead of re-deriving.

## What models do inside the product

The bake-off (P0-LLM and P0-GATE) decides these. Do not hard-code a model in code. Model ids and prices live in config.
