# U0 research (verified 2026-11-09 with `go list -m -versions` and module source)

- go-fsrs: use `github.com/open-spaced-repetition/go-fsrs/v4` v4.0.0 (needs go 1.26, local is 1.26.3).
  Its README says FSRS v6 and its default weights hold 35 numbers. `/v3` v3.3.1 has 19 weights, so it is
  an older FSRS. PLAN.md asks for FSRS-6, so v4. Note v4 `Repeat` and `Next` return an error.
- pgx: `github.com/jackc/pgx/v5` v5.12.0.
- Gemini: `google.golang.org/genai` v1.77.0. The older `github.com/google/generative-ai-go` is not used.
- Anthropic: `github.com/anthropics/anthropic-sdk-go` v1.84.0.
- go.mod keeps modules only if imported, so `internal/deps/deps.go` (build tag `tools`) pins all four.
- Postgres 16 image: `postgres:16`. `gen_random_uuid()` is built in since PG13.
- Migrations: tiny Go runner (`cmd/migrate`, files embedded) instead of golang-migrate, so no extra dependency.
- Unverified: SDK call signatures (U2 and U3 check them). Docker daemon version here is 28.5.1.
