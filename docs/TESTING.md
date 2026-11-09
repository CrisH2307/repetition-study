# Test plan for Dino

Your job is to decide whether the demo really works. You do not need to read code. Agents write the reports, you check them and say pass or fail. Never write "pass" without a number or a screenshot next to it.

What we can and cannot prove today: the loop works (a wrong explanation changes the next review and next week's plan). We cannot prove students learn more. Never say otherwise.

## Part A. Set up your machine (do this first, 30 minutes)

You need Docker Desktop (open and running), Go 1.26 or newer, Git, and Chrome.

```bash
git clone https://github.com/CrisH2307/repetition-study && cd repetition-study
git switch qa && git pull
go version
```

Then:

```bash
make up migrate test
```

Pass: the last lines all say `ok` or `no test files`, and nothing says `FAIL`. If anything fails, copy the full output to Cris and stop. Do not try fixes.

Rules:
- Test on the `qa` branch only. Run `git status` first. If it lists changed files, ask Cris.
- API keys go in a file called `.env` in the repo folder. Ask Cris for the keys. Never paste a key into chat, a doc or git.
- A line that says SKIP is not a pass. Tell Cris which test was skipped.
- Run every check twice. If the second run gives a different answer, that is a bug.

## Part B. Checks you can do now (no screen needed)

| # | Check | How | Pass |
| --- | --- | --- | --- |
| B1 | All automated tests | `make test` | No FAIL, run twice |
| B2 | Race test | `go test -race ./...` | No FAIL |
| B3 | History cannot be edited (empty table) | `docker compose exec db psql -U repetition -d repetition -c "TRUNCATE review_events;"` | An error that says `review_events is append-only` |
| B4 | No secrets in git history | `git grep -n -e AIza -e sk-ant -e sk-or $(git rev-list --all)` | No output |
| B5 | Migrations safe to repeat | `make migrate` twice | Second run prints nothing |
| B6 | Student never sees source text | `go test ./internal/teach/...` | `ok`. This runs the 10 leak probes on fake models |

After the demo has run once, repeat B3 with a real row: `UPDATE review_events SET grade = 'good';` and `DELETE FROM review_events;` must both fail the same way.

## Part C. The Hour 3 gate (most important)

The gate decides whether the checker may change the schedule. It uses the 20 explanations that you and Cris label by hand (about 10 correct, 5 partly correct, 5 wrong).

Before the gate runs:
1. Label your sheet alone. Do not look at Cris's labels. Labels are `supported`, `partial`, `contradicted`, `absent`.
2. Run `scripts/check_labels.py` with Cris. Settle each disagreement and save the result as `eval/data/labels_final.csv`. After this file is final, nobody edits it to make a result pass.

The gate bars (from PLAN.md):

| Bar | Pass |
| --- | --- |
| Wrong explanations flagged contradicted or absent | At least 4 of 5 |
| Correct explanations wrongly flagged contradicted | At most 1 of 10 |
| Corrections that have a valid citation | 100% |
| Facts the AI student states that it was never taught, 10 probes | 0 |

Read the report in `eval/results/gate.md`. Check that it shows real numbers for each model, the cost per session, and every failed example in full. Read the failed examples yourself, not just the count.

If a bar fails, stop and decide with Cris. Plan: narrow the concept list and switch the checker model once. If it still fails, gaps become suggestions only and do not change the schedule. We say that in the demo.

## Part D. The demo run (after the UI is merged into qa)

Do this by hand in Chrome, with a screenshot for each line. Use a fresh database: `docker compose down -v` then `make up migrate seed`.

| # | Do | Expect |
| --- | --- | --- |
| D1 | Add two subjects with weekly hours, level and exam date | A weekly plan appears |
| D2 | Open the plan, check each block shows a teach-back ahead of time | Teach blocks are visible |
| D3 | Teach one concept by voice. Plant one mistake on purpose | The AI student asks one question per turn and never corrects you |
| D4 | Finish the session | A report with a cited gap for the planted mistake. The quote comes from the course unit |
| D5 | Look at that concept's next review date | It moves earlier, and the change is visible |
| D6 | Open the cards | The gap made at least one card, and you can edit it |
| D7 | Open Further Planning | Recall and explanation are shown separately, next week's hours changed |
| D8 | Open the daily digest | One line per subject with due count and minutes |
| D9 | Move a review date by hand | It saves, and a manual override appears in the history |
| D10 | Say a correct explanation, no mistakes | No false gap, and the date does not move earlier |

D3 to D5 are the product. If they fail, nothing else matters.

## Part E. Things that go wrong on stage

| Risk | Test |
| --- | --- |
| Spoken math comes out garbled | Say a formula aloud and read the transcript. Use the edit box and the LaTeX box. Plan to explain in words on stage |
| Microphone and Chrome | Use Chrome only. Test the microphone permission on the demo laptop. Chrome speech sends audio off the device, so say that out loud |
| Venue wifi fails | Turn wifi off and watch what happens. Keep a recorded run of D1 to D9 as backup |
| API key missing or out of credit | Run D3 with a wrong key and confirm there is a clear error, not a blank screen |
| Slow answers | Time first word and the report on real models, 10 turns. These include the vendor's delay, so do not compare with fake-model numbers |

## Part F. Writing a bug

One bug per message. Use this shape:

- What I did (steps, in order)
- What I expected
- What happened (paste the error text or a screenshot)
- How bad: blocks the demo, wrong but workable, or cosmetic

## Order of work

1. Part A now.
2. Part B once Cris merges the unit branches into `qa`.
3. Label your sheet (Part C) as soon as the draft explanations exist. This is the longest task, start early.
4. Part C gate run, then stop and decide.
5. Part D and E after the UI and wiring are merged. The final run must pass on `qa` before anything goes to `main`.
6. Dino reports pass or fail to Cris. Cris pushes `qa` and `main`.
