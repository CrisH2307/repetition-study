-- Contract v4 (Phase 2, D36 to D44). Additive: new tables and nullable columns.
-- The one behaviour change is D43: the review_events trigger now lets a single
-- guarded delete-account path remove one user's events. No rows change here.

-- Weekly feedback (D40). One answer per user per week; never shown to others.
CREATE TABLE feedback_answers (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    uuid NOT NULL REFERENCES users(id),
    week_start date NOT NULL,
    question   text NOT NULL,
    answer     text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (user_id, week_start)
);

-- Consent record (D44): which version of the plain-words text the user accepted.
CREATE TABLE consents (
    user_id      uuid NOT NULL REFERENCES users(id),
    text_version text NOT NULL CHECK (text_version <> ''),
    accepted_at  timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, text_version)
);

-- CSV import (D38). Good rows become claims through the normal quote check;
-- bad rows are listed by row number in row_errors ([{"row":2,"error":"..."}]).
CREATE TABLE import_jobs (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    subject_id    uuid NOT NULL REFERENCES subjects(id),
    filename      text NOT NULL,
    status        text NOT NULL DEFAULT 'running' CHECK (status IN ('running', 'done', 'failed')),
    rows_total    int NOT NULL DEFAULT 0 CHECK (rows_total >= 0),
    rows_imported int NOT NULL DEFAULT 0 CHECK (rows_imported >= 0 AND rows_imported <= rows_total),
    row_errors    jsonb NOT NULL DEFAULT '[]' CHECK (jsonb_typeof(row_errors) = 'array'),
    created_at    timestamptz NOT NULL DEFAULT now(),
    finished_at   timestamptz,
    CHECK ((status = 'running') = (finished_at IS NULL))
);
CREATE INDEX import_jobs_subject_idx ON import_jobs (subject_id);

-- Model spend per calendar month (UTC), so the D33 cap survives restarts.
CREATE TABLE model_spend (
    month      date PRIMARY KEY CHECK (extract(day FROM month) = 1),
    usd        numeric NOT NULL DEFAULT 0 CHECK (usd >= 0),
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- Dispute outcome (D39). Upheld disputes carry the label a person says is right
-- and become dev-split eval cases. Old rows keep nulls.
ALTER TABLE disputes
    ADD COLUMN outcome        text CHECK (outcome IN ('upheld', 'rejected')),
    ADD COLUMN expected_label text CHECK (expected_label IN ('supported', 'partial', 'contradicted', 'absent')),
    ADD CONSTRAINT disputes_outcome_reviewed CHECK (outcome IS NULL OR status = 'reviewed'),
    ADD CONSTRAINT disputes_upheld_label CHECK ((outcome IS NOT DISTINCT FROM 'upheld') = (expected_label IS NOT NULL));

-- D43. DELETE is allowed only for rows of the user named by
-- `SET LOCAL app.delete_user = '<user id>'` in the same transaction (the row
-- reaches its user through concept and subject). UPDATE, TRUNCATE and every
-- other DELETE stay rejected. Only the delete-account path may set it, and only
-- with SET LOCAL, so the setting ends with the transaction.
CREATE OR REPLACE FUNCTION review_events_reject_change() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE
    target text := current_setting('app.delete_user', true);
BEGIN
    IF TG_OP = 'DELETE' AND coalesce(target, '') <> '' AND EXISTS (
        SELECT 1 FROM concepts c JOIN subjects s ON s.id = c.subject_id
        WHERE c.id = OLD.concept_id AND s.user_id::text = target
    ) THEN
        RETURN OLD;
    END IF;
    RAISE EXCEPTION 'review_events is append-only (% rejected)', TG_OP
        USING ERRCODE = 'restrict_violation';
END;
$$;
