# Re-running the checker on upheld disputes (for Cris)

This is paid and runs on the development split only (D39). The frozen test split and its labels never change. Not run against a real model; unverified.

1. Export disputes: `DATABASE_URL=... go run ./cmd/export -disputes > disputes.csv`. That file has no student text or labels yet. Build a CSV (or a JSON list of the Go `DisputeCase` type from `UpheldDisputes`) with columns `dispute_id, concept_id, claim_id, student_text, expected_label, outcome`. Only `outcome = upheld` rows with one of supported, partial, contradicted, absent are kept.
2. Convert: `python3 scripts/disputes_to_eval.py disputes.csv`. It writes `content/private/eval/disputes.jsonl` (gitignored, because the student's words are user data) and lists skipped rows.
3. Run, small cap first:

```bash
go run ./cmd/evalcheck --models "$CHECKER_MODEL" --suites human --split dev --disputes content/private/eval/disputes.jsonl --max-cost 1
```

`--disputes` with `--split test` is refused. The result is a DEV RUN: numbers only, no pass or fail. Each case adds one explanation labelled for its one disputed claim; other claims in that concept are not scored.
