# P3-CONTRACT research

- Phase 2 schema ends at 0005; users has no email, so cmd/instructor must find a user by id or name.
- Delete account (D43) deletes subjects then users; new tables use ON DELETE CASCADE so that path needs no change.
- /v1/me extras are composed in internal/app (like consent_version), so `plan` is a spec field only. `is_instructor` lives on domain.User.
- Stripe specifics (signature format, Checkout modes) are P3-BILLING's to verify against current docs. Unverified here.
