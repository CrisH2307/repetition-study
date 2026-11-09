---
name: unit
description: Run one build unit of the Space Repetition project from its prompt file, using Research, Plan, Implement.
disable-model-invocation: true
argument-hint: [U0-U7, GATE, INT, QA or RELEASE]
allowed-tools: Bash(git *)
---

Unit: $0
Current branch: !`git branch --show-current`
Recent commits: !`git log --oneline -5 2>/dev/null || true`

Your task is in rpi/$0/PROMPT.md. Read it first, then rpi/$0/HANDOFF.md and rpi/$0/impl-log.md if they have content. If they do, resume from there and do not repeat finished work.

Work in this order:

1. Research: verify versions, library APIs and anything the prompt leaves open. Keep rpi/$0/research.md under 30 lines.
2. Plan: write steps, files, tests and risks in rpi/$0/plan.md. Continue without asking unless the prompt says to wait for Cris, or the work conflicts with CLAUDE.md or the contract.
3. Implement in small commits, with tests alongside the code. After each step, add one line to rpi/$0/impl-log.md: what changed and the test result.
4. Push the branch and open small PRs with `gh pr create --base qa`, one concern each. Never merge. Cris merges.
5. Finish with the report CLAUDE.md asks for.

Stop and ask Cris if a test keeps failing after two different fixes, or if you need a file outside this unit's boundary.
