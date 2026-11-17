# P2-WEB2 plan
1. api.ts: v4 calls. Pilot.tsx: Consent, Guide, FeedbackCard, Settings. App.tsx: consent gate, Settings link, guide and question on Today.
2. Course.tsx: empty states, CSV import with polling and per-row results.
3. mock/plugin.ts: v4 endpoints. Tests: vitest consent, e2e/pilot.mock.ts (consent, guide, question, empty states, import, export, delete), loop.e2e.ts consent skip.
Risk: consent remembered per browser only (see research.md).
