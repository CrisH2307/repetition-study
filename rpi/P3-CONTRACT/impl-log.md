# P3-CONTRACT impl log

- 0006 migration plus TestContractV5Checks: applied on a pg_dump copy of the local Phase 2 db; race suite green on it.
- domain v5, spec v5, fakes v5, skip list: build, vet, `go test -race ./...` green.
- Billing removed on Cris's call (subscriptions later, once people use the product): no entitlements or billing_events tables, no Billing interfaces, no /v1/billing paths, no `plan` on /v1/me. Re-tested green on a fresh copy.

## Interfaces for wave 2 (internal/domain/v5.go)

- P3-SHARE: `ShareStore` (CreateShare, RevokeShare, ShareByToken: ErrNotFound when revoked, SetInstructor) and `Sharing` (Create, Revoke, Preview, Import returning `PackImport` with `NeedsQuote`). Claims without a quote are skipped and listed, not saved as drafts. Token hash is 32 bytes (DB check).
- P3-API: `is_instructor` comes on domain.User once the store selects it. Remove the 4 v5 entries from `notServed`.
- Not in the contract: a list of my share links (only create and revoke). Ask Cris if the web needs it.
