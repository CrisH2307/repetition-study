# P1-AUTH research

- Go 1.26.3. `net/http.CrossOriginProtection` (Go 1.25+) does the same-origin check for unsafe methods using Sec-Fetch-Site, then Origin vs Host. Requests with neither header pass (curl, tests). Verified by `TestCrossOriginPostRefused`.
- `crypto/rand.Read` never returns an error since Go 1.24 (crashes the program instead).
- Contract: `domain.Auth`, `domain.AuthStore` in internal/domain/v3.go; fake `fakes.AuthStore`. `RedeemInvite` returns the same user on every use, so a reused link signs that user in again and never creates a second account.
- Error body shape `{"error": msg}` matches internal/app/handlers.go.
- Store implementation is P1-STORE (not merged); cmd/invite asserts at run time until then.
