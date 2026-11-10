---
name: handoff
description: Save this unit's state so a new session can continue it, when context usage passes 50 percent.
disable-model-invocation: true
argument-hint: [unit]
---

Stop new work and write rpi/$0/HANDOFF.md containing:
- current state and what works, with test results
- decisions made and why
- open questions
- exact next steps
- files, branches and open PRs

Then tell me to start a new session and run `/unit $0`, which resumes from this file.
