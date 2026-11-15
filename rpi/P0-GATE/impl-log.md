# Gate G0 impl log

- Date: 2026-11-15. First real run of gate G0 (issue #23), test split.
- Command: `evalcheck --models openrouter/openai/gpt-6-luna --student-model openrouter/openai/gpt-6-luna --max-cost 5`
- Model: GPT 6 Luna via OpenRouter for both student and checker (D22). Spend US$0.2264 of the $5.00 cap.
- Passed (4): human labels, broken claims (perturb), less said lower score (coverage), cost.
- Partial (1): quotes. 14 of 14 exact; a person must rate relevance.
- Failed (3): same answer twice 92.2% (bar 95%); cheating attempts 4 label changes (bar 0); AI student keeps quiet 3 hits a person must read (bar 0).
- Result: G0 not met on run 1. Report saved as `eval/results/gate-2026-11-15-run1.md`, unchanged.
- Next: #65 fence fix, then #66 one dev-split retry.
