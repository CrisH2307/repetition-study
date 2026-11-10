# P0-STORE plan
1. `internal/store/store.go`: `Store` implements `domain.TeachStore` and `domain.PackLoader` with parameterised pgx SQL.
2. `LoadPack`: one transaction, upsert by id, history insert-only (`ON CONFLICT DO NOTHING`).
3. `store_test.go`: one conformance function run on `teach.MemStore` and `Store`; Postgres tests for restart, labels, source filtering, pack idempotence.
Risks: no pool yet (go.mod); a `SaveResults` correction without citation errors (table CHECK), the engine must not send one.
