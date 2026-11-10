---
name: status
description: Report build progress against the 5pm deadline and recommend cuts when behind.
disable-model-invocation: true
allowed-tools: Bash(git *) Bash(date *)
---

Time now: !`TZ=America/Toronto date '+%H:%M %Z'`
Worktrees: !`git worktree list`
Deadline: 5pm America/Toronto.

Read rpi/*/impl-log.md in each worktree and `git log origin/qa`. Tell me in a table what is done, broken and not started.
If we are behind, recommend cuts in this exact order: digest delivery, plan editing, generated cards. Never cut the checker or the schedule update. Wait for my decision before changing anything.
