You own P0-GATE on branch feature/p0-gate, from qa after P0-LLM, P0-EVALDATA and
P0-CITE are merged. Owners: Cris runs, Dino reads, both decide. Files:
eval/results/, rpi/P0-GATE/. Do not change code or data. This replaces rpi/GATE
for forward work.

Read first: docs/ROADMAP.md (Gate G0 table), docs/DECISIONS.md, docs/STATUS.md.
Decisions: D9 (bars) and D11 (models, budget), D24. If D9 is Pending, report
against the Decided hackathon bars and show the proposed bars as "proposed".
Write plan.md, then STOP until Cris approves it (regulation 5).

Steps
1. Run cmd/evalcheck on the frozen test split for each model D11 lists, within
   the budget cap. Never use the test split for tuning.
2. Run the leak mode on the 30 probes with the real student model.
3. Run stability (3 runs), injection, coverage gradient and perturbation suites
   that have data; mark missing suites "not run" with the reason.
4. Write eval/results/gate.md: every bar, actual number, pass or fail, cost per
   session, latency, model ids, dataset hash, commit hash, every failed example
   in full. Add the P0-INT loop result and Dino's session notes when available.

Tests: none; this unit adds no code. Done when eval/results/gate.md exists with
every bar above filled in or marked "not run" with a reason.

Do not tune prompts or edit labels to pass. Then stop and ask Cris and Dino what
to do; the G0 decision is theirs.

Report in under 200 words: numbers against bars, cost, what was not run.
