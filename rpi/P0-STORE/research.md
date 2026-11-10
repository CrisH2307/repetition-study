# P0-STORE research
- pgx v5.12.0 is in go.mod. `pgxpool` needs `github.com/jackc/puddle/v2` (go.mod change, outside this unit). The store takes a small `DB` interface, so a pool or a single conn both work. P0-INT must run `go mod tidy` when it wires a pool.
- `sources` has no subject column. "Approved for the subject" = kind `course` and cited by a claim of a concept in that subject. Allowlist sources are dropped (CLAUDE.md). `reviewed_by` is not required, because the draft pack is unreviewed. Unverified: Cris may want review required later.
- `LoadConcept` returns required claims only (matches the interface comment).
- Tests use a throwaway schema (search_path) in the dev database, dropped on cleanup. No rows land in `public`.
