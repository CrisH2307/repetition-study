You own integration on branch feature/integration, from qa. Files: cmd/, wiring
code, Makefile. If a unit has a bug, report it and open a fix/<name> branch from qa.
Fix one bug per PR.

Replace fakes with real implementations in this merge order: scheduler, planner,
checker, teach engine, digest, UI. After each, run make test.
Load content/unit.json, content/allowlist.json and content/seed.sql with make seed.

Prove the whole loop with one automated test using the fake LLM, then once with
real models:
seed; teach the demo concept with one planted mistake; finish the session;
assert a verified citation, an earlier next review, a teach_back event, a new gap
card, more hours next week for that subject and a digest line.

Report in under 200 words, with anything that deviates from the contract.
