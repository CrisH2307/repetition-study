# Running gate G0 (for Cris)

This is the exact run for the gate in `docs/PLAN.md`. It spends real money, so do not start it until you have approved the spend. Nothing here was run against a real model when this page was written, so every step that needs a real model is **unverified**. The flags were copied from `go run ./cmd/evalcheck --help`.

## 1. Before you start

- Frozen labels are in place: `eval/data/labels_final.csv`, recorded in `eval/data/FROZEN.md` (80 rows, kappa 0.964, 2 settled). `evalcheck` only scores them if the SHA-256 in `FROZEN.md` matches the file. Do not edit them.
- The real course unit must exist at `content/private/unit.json` with the source text filled in (see `content/README.md`). `content/private/` is gitignored. Without it, `evalcheck` stops with a clear message.
- Put your keys in `.env` (gitignored), names only here:

| Name | Used for |
| --- | --- |
| `GEMINI_API_KEY` | the AI student (leak suite) |
| `STUDENT_MODEL` | the AI student model id, a `gemini-*` id |
| `ANTHROPIC_API_KEY` | the checker |
| `CHECKER_MODEL` | the checker model id, for example `claude-sonnet-5-5` |

To use OpenRouter instead (D21), with GPT 6 Luna for both roles (D22), set these and leave the four above out. The prepaid balance is the hard limit, and `--max-cost` still applies:

| Name | Value |
| --- | --- |
| `OPENROUTER_API_KEY` | your OpenRouter key |
| `STUDENT_MODEL` | `openrouter/openai/gpt-6-luna` |
| `CHECKER_MODEL` | `openrouter/openai/gpt-6-luna` |

The id is priced in `models.json` (prices from OpenRouter's public model list, 2026-11-14). The command in section 3 is unchanged. Not run against the live OpenRouter API, so unverified.

`evalcheck` reads these from the shell environment and does not read `.env` itself (`make` does). Load the file first:

```bash
set -a; . ./.env; set +a
```

## 2. Prices and the spend cap

`internal/llm/models.json` lists prices in US dollars per million tokens. Today it has one entry: `claude-sonnet-5-5` at 2 in and 10 out. Its source note says it is unverified, so check it against Anthropic's pricing page before you trust any cost figure.

There is no entry for a Gemini model. `evalcheck` refuses to run a model with no price while `--max-cost` is above 0 (it says so and exits). Pick one:
- add a verified price for your `STUDENT_MODEL` to `models.json` (commit it on its own branch first), or
- pass `--max-cost 0`, which means no cap. Not recommended.

Every check asks the checker model 3 times and keeps the majority label per claim (D23), so a run makes about 3 times as many checker calls as before and costs about 3 times as much on the checker side. The default cap is 5 US dollars across all models and suites together. The run stops before it passes the cap and still writes what it measured. Whether Gemini accepts the thinking setting of 0 in `models.json` is also unverified.

## 3. The command

Start small. Run one suite first to check keys and cost:

```bash
go run ./cmd/evalcheck --models "$CHECKER_MODEL" --suites human --max-cost 1
```

Then the full gate:

```bash
go run ./cmd/evalcheck --models "$CHECKER_MODEL" --student-model "$STUDENT_MODEL" --max-cost 5
```

Other flags (defaults in brackets): `--suites` [all 8: human, perturb, citations, stability, injection, coverage, leak, cost], `--split` [test; also dev or all], `--turns` [8, turns in each full session of the cost row], `--unit` [content/private/unit.json], `--data` [eval/data], `--labels` [eval/data/labels_final.csv]. Do not tune prompts on the test split to pass (rule in `docs/PLAN.md`).

In the human labels row, an explanation where the three checker votes have no majority is still scored, in the strictest way: every claim in it counts as a miss (a correct claim as marked contradicted, a wrong claim as not flagged, a missing claim as not caught), and the headline says how many were unchecked.

A dry run with no cost, to check the tool itself: `--models fake --suites human --unit content/unit.json`. This was not re-run for this page.

## 4. Where the output lands

| File | Committed? | What it is |
| --- | --- | --- |
| `eval/results/gate.md` | yes, if you choose | The report. It shows only text from committed data, never course text. Dino reads this |
| `content/private/eval/gate_private.md` | never (gitignored) | Raw checker output, quotes, every leak reply. Read every failed example here |
| `content/private/eval/citations_to_rate.csv` | never (gitignored) | The sheet for the human citation check |

## 5. After the run: the human citation check

The bar is at least 90 percent of quotes rated relevant by a person. Code only proves a quote is exact. A person (Dino) opens `content/private/eval/citations_to_rate.csv` and, for each row, writes `yes` or `no` in the last column, `relevant`, for whether the quote supports the correction. Count it and put the number in the report by hand. The tool does not read the sheet back (unverified: no code reads it).

The leak row also needs a person: code only catches copied runs of 5 or more words, so someone must read the AI student replies in `gate_private.md`.

## 6. If a row fails

Stop and decide with Cris. One iteration is allowed on the dev split (`--split dev`), as in `docs/PLAN.md`. Nobody edits `labels_final.csv`.
