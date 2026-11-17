# P3-CONTRACT impl log

- 0006 migration plus TestContractV5Checks: applied on a pg_dump copy of the local Phase 2 db (7 users); race suite green on it.
- domain v5, spec v5, fakes v5, skip list: build, vet, `go test -race ./...` green.

## Interfaces for wave 2 (internal/domain/v5.go)

- P3-BILLING: `BillingStore` (Entitlement, UserByCustomer, ApplyEvent: event and entitlement in one tx, false when the event id was seen) and `Billing` (Status, CheckoutURL, HandleWebhook returning `ErrBadSignature`, Entitled: true while billing is off). Month needs a subscription id, term must have none (DB check).
- P3-SHARE: `ShareStore` (CreateShare, RevokeShare, ShareByToken: ErrNotFound when revoked, SetInstructor) and `Sharing` (Create, Revoke, Preview, Import returning `PackImport` with `NeedsQuote`). Chosen: claims without a quote are skipped and listed, not saved as drafts. Token hash is 32 bytes (DB check).
- P3-API: /v1/me adds `plan` from Billing.Status; `is_instructor` comes on domain.User once the store selects it. Webhook route is open (`security: []`), reads `Stripe-Signature`. Remove the 7 v5 entries from `notServed`.
- Not in the contract: a list of my share links (only create and revoke). Ask Cris if the web needs it.
