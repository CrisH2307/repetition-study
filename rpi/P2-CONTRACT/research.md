# P2-CONTRACT research

- Dev DB is at Phase 1 (0001 to 0004 applied). Copy made as `p2_contract_copy` with pg_dump; 0005 applied on it, second run is a no-op.
- `current_setting('app.delete_user', true)` returns NULL when never set and '' after a transaction that set it ends, so the trigger treats both as "not set".
- `set_config(name, value, true)` is the same as `SET LOCAL` and takes a bind parameter; plain SET LOCAL cannot.
- A TRUNCATE trigger has TG_OP 'TRUNCATE', so the D43 branch never matches it.
- The D33 cap today: `internal/app/owner.go` capLLM over the in-memory meter (resets on restart).
- Disputes have only open/reviewed; D39 needs an outcome and the right label, added as nullable columns.
