-- Contract v5 (Phase 3, D49 to D51). Additive only: new tables and one column
-- with a default. No existing row changes meaning. New rows cascade on user
-- and course delete, so the D43 delete-account path needs no change.

-- D51. Set by a command Cris runs, never by the user.
ALTER TABLE users ADD COLUMN is_instructor boolean NOT NULL DEFAULT false;

-- D49. One entitlement row per user. A term pass is a one-time payment, so it
-- has no subscription id.
CREATE TABLE entitlements (
    user_id         uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    plan            text NOT NULL CHECK (plan IN ('month', 'term')),
    status          text NOT NULL CHECK (status IN ('active', 'past_due', 'canceled')),
    period_end      timestamptz NOT NULL,
    customer_id     text NOT NULL CHECK (customer_id <> ''),
    subscription_id text CHECK (subscription_id <> ''),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    CHECK ((plan = 'month') = (subscription_id IS NOT NULL))
);
CREATE UNIQUE INDEX entitlements_customer_idx ON entitlements (customer_id);

-- D49. Every webhook event once, keyed by the provider's event id, so a
-- replayed or retried event changes nothing.
CREATE TABLE billing_events (
    event_id    text PRIMARY KEY CHECK (event_id <> ''),
    event_type  text NOT NULL CHECK (event_type <> ''),
    received_at timestamptz NOT NULL DEFAULT now(),
    payload     jsonb NOT NULL CHECK (jsonb_typeof(payload) = 'object')
);

-- D50, D51. A share link for one course. Only the SHA-256 of the token is
-- stored. includes_sources is for instructor packs only; code checks
-- users.is_instructor when the link is made.
CREATE TABLE shared_packs (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id         uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    subject_id       uuid NOT NULL REFERENCES subjects(id) ON DELETE CASCADE,
    token_hash       bytea NOT NULL UNIQUE CHECK (length(token_hash) = 32),
    includes_sources boolean NOT NULL DEFAULT false,
    created_at       timestamptz NOT NULL DEFAULT now(),
    revoked_at       timestamptz,
    CHECK (revoked_at IS NULL OR revoked_at >= created_at)
);
CREATE INDEX shared_packs_owner_idx ON shared_packs (owner_id, subject_id);
