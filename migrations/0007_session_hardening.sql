-- P3-HARDEN. Additive. An invite can be revoked; a session remembers when it
-- was last used so an idle one can expire (14 days).
ALTER TABLE invites ADD COLUMN revoked_at timestamptz;
ALTER TABLE auth_sessions ADD COLUMN last_seen_at timestamptz; -- null until first use, so the store clock decides
