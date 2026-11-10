# How to label the explanations (for Dino and the classmate)

You each label the same 80 rows on your own. Do not look at the other person's
sheet and do not talk about the answers until you have both finished.

## What one row is

Each row has a student explanation and one claim from the course. You decide
how well that explanation covers that one claim. Pick exactly one label:

| Label | Use it when |
| --- | --- |
| `supported` | The explanation says the claim correctly and completely, including its conditions (for example "independent"). Other words or other notation are fine. |
| `partial` | Part of the claim is there and nothing said about it is wrong, but a piece is missing or vague (a condition, a step, the exact formula). |
| `contradicted` | The explanation says something about this claim that is wrong: a wrong formula, a wrong sign, a swapped parameter, a wrong condition. Use this even if other parts are right. |
| `absent` | The explanation does not talk about this claim at all. |

If you are torn between two labels: something wrong beats something missing
(`contradicted` over `partial`), and ask yourself "could a classmate learn the
whole claim from this text?" (yes means `supported`, no means `partial`).

## One example of each label

Claim: "If T ~ Exp(λ), then P(T > t) = e^{-λt} for t >= 0."

- `supported`: "An exponential time with rate λ is still waiting at time t with probability e^{-λt}."
- `partial`: "The chance that T is still waiting at time t shrinks exponentially." (no rate, no formula)
- `contradicted`: "P(T > t) = 1 - e^{-λt}." (that is P(T <= t))
- `absent`: "The mean of T is 1/λ." (true, but about something else)

## Steps

1. Open Terminal, go to the project folder, and get the newest files:
   `git switch qa` then `git pull`.
2. Open your sheet: Dino uses `eval/data/labels_dino.csv`, the classmate uses
   `eval/data/labels_classmate.csv`. Google Sheets works: File, Import, Upload.
3. For each row, read the `explanation` and the `claim`, then type one of the
   four labels in the `label` column, in lower case. Use `note` for anything
   you were unsure about (optional).
4. Do not change any other column. Do not add, delete or sort rows.
5. Save as CSV with the same file name (Google Sheets: File, Download,
   Comma-separated values) and put it back in `eval/data/`, replacing the old file.
6. When both sheets are done, run:
   `python3 scripts/check_labels.py`

## What pass looks like

- The script prints "You agree on N of 80 claim labels" and a kappa number.
- If it lists disagreements, sit together, read each one, and type the label
  you both agree on in the `label` column of `eval/data/labels_resolved.csv`
  (the script creates that file for you).
- Then run `python3 scripts/check_labels.py --freeze`. Pass is the line that
  starts with "Frozen." Tell Cris; Cris commits the files.

## If something goes wrong

- "Not ready to compare": fix the rows it lists (empty or misspelt labels), save, run again.
- Anything else, including "Not frozen: kappa is ... below 0.6", or an error
  you do not understand: copy the whole output to Cris and stop.
