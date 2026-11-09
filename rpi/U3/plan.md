# U3 plan

1. internal/checker/verify.go: whitespace normalize, citation check, hint overlap, label rules. Tests: correct quote, paraphrase, unapproved source, injection, hint overlap, absent handling.
2. internal/checker/checker.go: prompt (transcript fenced as untrusted), LLM.JSON call, CheckDetailed returns stats for eval, Check wraps it. `AllowAbsent` is the finish-mode switch.
3. cmd/evalcheck: loaders, provider adapters (gemini, claude, fake), metering, report writer to eval/results/<model>.md. Tests on aggregation and report with fake LLM.
Risks: real SDK paths untested, data formats assumed, injection is only partly code-enforceable (the model still labels).
