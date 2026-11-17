# Plan
1. cmd/loadtest: stdlib tool, host guard, setup (sign in, one course with 3 concepts), timed GET loop, p50/p95/p99, exit 1 if bar missed. Unit test for guard and percentile.
2. Run locally at 500 users on the fake model; record.
3. docs/SECURITY.md checklist, docs/RESTORE.md drill.
Risk: laptop shares CPU with the test, so numbers are pessimistic.
