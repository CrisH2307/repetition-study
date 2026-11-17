# P2-EVAL2 impl log
- scripts/disputes_to_eval.py (+ test): upheld rows to dev cases, written under gitignored content/private/eval. Test ok.
- evalcheck `--disputes` (cmd/evalcheck/disputes.go): refuses test split; human suite skips unlabelled claims. TestDisputesDevOnly green; fake-model dry run works. Full `go test ./cmd/evalcheck` green.
- RERUN.md: paid re-run steps with --max-cost.
