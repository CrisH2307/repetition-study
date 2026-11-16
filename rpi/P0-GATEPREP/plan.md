# P0-GATEPREP plan (issue #47)

1. New sets from explanations.jsonl only: 10 injection attacks, coverage gradient (4 concepts x 100/75/50/25/0%), 30 leak sessions. `eval/data/make_gate_sets.py`.
2. evalcheck runs the 8 G0 suites per checker model, one shared meter and spend cap for the whole run.
3. Each suite: PASS or FAIL only from a measurement; NOT RUN with a reason when data is missing, the cap stops it, or the checker labelled nothing; PARTIAL when a person still has to look (citation relevance, leak replies, student cost unpriced).
4. gate.md: summary for Dino, numbers per model, commit and dataset hashes, every failed example. Raw output that can hold course text goes to content/private/eval/.
5. Tests with scripted fake checkers, a must-fail case per suite, and a full llm.New("fake") dry run.
