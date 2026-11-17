# P2-MAIN plan

Scope and waves: `docs/PLAN.md` (Phase 2). Defaults D36 to D40. Starts when Phase 1 is in qa and Dino has passed it.
Units run like Phase 1: issue, worktree from origin/qa, PR into qa, auto-fix on, no attribution, Sonnet unless noted (P2-CONTRACT on Opus).

## Cris's answers (#107, 2026-11-16)
1. No ethics review, no study arms (D41). 2. No email digest (D42). 3. Delete everything, review history included (D43). 4. No separate legal read (D44, wording unverified).
Note: CLAUDE.md says "review_events is append-only". D43 adds one exception; CLAUDE.md is only edited with Cris's own words, so ask him to approve a one-line note there.

## Status
| Unit | Issue | PR | State |
| --- | --- | --- | --- |
| P2-CONTRACT | #118 | #120 | merged |
| P2-STORE | #121 | #129 | merged |
| P2-METRICS | #122 | #126 | merged |
| P2-PRIVACY | #123 | #130 | merged |
| P2-IMPORT | #124 | #127 | merged |
| P2-EVAL2 | #125 | #128 | merged |
| P2-API | #131 | #134 | merged |
| P2-WEB2 | #132 | #133 | merged |
| Release to main | | #139 | merged and deployed (Vercel plus Fly.io Toronto) |
| Test | #138 | | open: Dino tests the live site |
| Finish leftovers | #140 | | this PR |
