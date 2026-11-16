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

1. Download your CSV file from Cris (Dino: `labels_dino.csv`, classmate:
   `labels_classmate.csv`) and open it in Google Sheets: File, Import, Upload.
2. For each row, read the `explanation` and the `claim`, then type one of the
   four labels in the `label` column, in lower case. Use `note` for anything
   you were unsure about (optional).
3. Fill in only the `label` and `note` columns. Do not change any other column.
   Do not add, delete or sort rows.
4. Math may show as symbols in Sheets. That is fine, because only the `label`
   and `note` columns are used.
5. Download it again: File, Download, Comma-separated values (.csv). Keep the
   same file name and send the file to Cris. Do not share your answers with the
   other labeller.

## What Cris does

1. Put the two files in `eval/data/`, then run `python3 scripts/check_labels.py`.
2. If it lists disagreements, the two labellers sit together, read each one,
   and type the label they both agree on in the `label` column of
   `eval/data/labels_resolved.csv` (the script creates that file).
3. Run `python3 scripts/check_labels.py --freeze`. Pass is the line that
   starts with "Frozen." Cris commits the files.

## What pass looks like

- The script prints "You agree on N of 80 claim labels" and a kappa number.
- The freeze step prints "Frozen."

## If something goes wrong

- "Not ready to compare": fix the rows it lists (empty or misspelt labels), save, run again.
- Anything else, including "Not frozen: kappa is ... below 0.6", or an error
  you do not understand: copy the whole output to Cris and stop.
