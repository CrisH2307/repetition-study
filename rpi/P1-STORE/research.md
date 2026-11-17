# P1-STORE research

- Schema: migrations 0001 to 0004. Interfaces: internal/domain/v3.go. Fakes (internal/fakes/v3.go) used as the behaviour reference.
- pgx v5.12.0, `DB` interface already in store.go. New code takes a small `q` interface so pool and tx share helpers.
- Recall and explanation use the same definitions as internal/planner/progress.go (good/easy share; required claims supported in the latest labelled session).
- Unverified: real DB with Phase 0 data (tests use a clean schema only).
