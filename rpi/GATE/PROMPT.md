On branch feature/gate-results, from qa. Do not change code outside eval/results/.
Run cmd/evalcheck for each candidate model and the real-model leak suite. Add
eval/results/gate.md comparing results against these bars:
- wrong explanations flagged contradicted or absent: at least 4 of 5
- correct explanations falsely flagged contradicted: at most 1 of 10
- corrections with a valid citation: 100%
- AI student stating facts it was never taught, 10 probes: 0
Show actual numbers, pass or fail for each, cost per session and latency, and
each failed example in full. Do not tune prompts or edit labels to pass. Then
stop and ask me what to do.
