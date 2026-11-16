-- Contract v2. Additive only: one new table, new columns, one index. No data changes.
-- ADD COLUMN does not fire row triggers, so review_events stays append-only.

CREATE TABLE source_blocks (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    source_id   uuid NOT NULL REFERENCES sources(id),
    page        int,
    ordinal     int NOT NULL,
    kind        text NOT NULL CHECK (kind IN ('text', 'math', 'heading', 'figure')),
    text_md     text NOT NULL,
    -- Written by code with internal/cite's normalizer, never by a model.
    text_norm   text NOT NULL,
    verified_by text NOT NULL DEFAULT '',
    verified_at timestamptz,
    UNIQUE (source_id, ordinal)
);

-- IANA zone name. Code checks it with time.LoadLocation before use.
ALTER TABLE users ADD COLUMN time_zone text NOT NULL DEFAULT 'America/Toronto';

ALTER TABLE review_events
    ADD COLUMN claim_id        uuid REFERENCES claims(id),
    ADD COLUMN session_id      uuid REFERENCES teach_sessions(id),
    ADD COLUMN self_grade      text CHECK (self_grade IN ('again', 'hard', 'good', 'easy')),
    ADD COLUMN checker_model   text,
    ADD COLUMN checker_version text;

-- The session id is the teach_back idempotency key: one session, one teach_back event.
CREATE UNIQUE INDEX review_events_teach_back_session_uniq
    ON review_events (session_id) WHERE event_type = 'teach_back';
