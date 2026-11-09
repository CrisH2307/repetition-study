# U0 plan

PR series into qa, one concern each, merge in this order:
1. feature/setup: starter kit, go.mod (pinned deps), compose.yaml, Makefile, .env.example, empty rpi files.
2. feature/setup-domain: internal/domain types and interfaces, internal/fakes with tests.
3. feature/setup-db: migrations, append-only trigger on review_events, cmd/migrate, seed, trigger test.
4. feature/setup-api: api/openapi.yaml, cmd/api with /healthz and a test.

Tests: fakes are deterministic (run twice, compare). Trigger test needs DATABASE_URL and skips without it,
`make test` sets a default. Done = `make up migrate test` green on a merged checkout.
Risks: contract gaps are listed in the final report (users table, Card type, Clock has no ctx).
