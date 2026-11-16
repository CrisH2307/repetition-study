# P1-MAIN plan

Scope, waves and units: `docs/PLAN.md` (Phase 1). Defaults: D26 to D34.

## Run order
1. This plan PR (chore/phase1-plan) merges first. Wave 1 units start from this branch so they have their prompts.
2. Wave 1: P1-CONTRACT (Opus, high), P1-E2E (Sonnet, medium), P1-IMAGE (Sonnet, low). Parallel.
3. Wave 2 after P1-CONTRACT merges: P1-STORE, P1-COURSE, P1-TODAY (Sonnet), P1-AUTH (Opus, security). Parallel.
4. Wave 3: P1-API (Sonnet, high), P1-WEB (Opus, medium). Parallel.
5. Wave 4: P1-DEPLOY (Sonnet) once Cris answers the question below, then a test issue for Dino on the deployed site.
6. After each wave merges into qa: a plain-language test issue for Dino; PR qa into main after he passes.

Review per PR: boundary vs `.claude/boundaries.json`, CI green, diff read; `/code-review medium` for P1-CONTRACT, P1-AUTH and P1-COURSE. Pass or fix as a PR comment.

## One question for Cris (asked 2026-11-16, issue #85)
1. Host (Canadian region, D17): Fly.io Toronto, DigitalOcean Toronto, or another. Prices unverified; checked before anything is paid.
2. Sign-in: keep invite links (free, D29), or Google sign-in, or email links (needs an email service).
3. Domain: the host's free address until the name is chosen (D16), or buy one now.
4. Pilot model budget: a monthly cap for checks (D33). Worst case per session is US$0.08, so 6 students at 4 sessions a week is about US$8 a month.

## Status
| Unit | Issue | PR | State |
| --- | --- | --- | --- |
| P1-CONTRACT | #87 | | starting |
| P1-E2E | #88 | | starting |
| P1-IMAGE | #89 | | starting |
