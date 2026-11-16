-- Contract v3 (Phase 1). Additive: new tables and nullable columns. Phase 0 rows
-- stay with the default user. The only data change fills sources.subject_id
-- where one subject's claims cite the source. No rows are removed.

-- Sign-in by invite link (D29). Tokens are random; only their SHA-256 is stored.
CREATE TABLE invites (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    token_hash bytea NOT NULL UNIQUE,
    name       text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    used_at    timestamptz,
    user_id    uuid REFERENCES users(id),
    CHECK ((used_at IS NULL) = (user_id IS NULL))
);

CREATE TABLE auth_sessions (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    token_hash bytea NOT NULL UNIQUE,
    user_id    uuid NOT NULL REFERENCES users(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL
);
CREATE INDEX auth_sessions_user_idx ON auth_sessions (user_id);

-- Courses are subjects. Archived instead of deleted, because review_events
-- (append-only) point at their concepts.
ALTER TABLE subjects ADD COLUMN archived_at timestamptz;
CREATE INDEX subjects_user_idx ON subjects (user_id);

-- Sources reach their user through the course.
ALTER TABLE sources ADD COLUMN subject_id uuid REFERENCES subjects(id);
UPDATE sources s SET subject_id = x.subject_id
FROM (
    SELECT cl.source_id, min(co.subject_id::text)::uuid AS subject_id
    FROM claims cl JOIN concepts co ON co.id = cl.concept_id
    GROUP BY cl.source_id
    HAVING count(DISTINCT co.subject_id) = 1
) x
WHERE s.id = x.source_id AND s.subject_id IS NULL;

-- Uploads and pages (D27). A PDF or pasted text, read into pages; the student
-- checks each page. Only Ready pages become source blocks of source_id.
CREATE TABLE uploads (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    subject_id  uuid NOT NULL REFERENCES subjects(id),
    filename    text NOT NULL,
    pdf         bytea CHECK (octet_length(pdf) <= 20 * 1024 * 1024),
    pasted_text text,
    status      text NOT NULL DEFAULT 'reading' CHECK (status IN ('reading', 'needs_checking', 'ready')),
    error       text NOT NULL DEFAULT '',
    -- The approved course source that Ready pages feed. Kind 'course'.
    source_id   uuid REFERENCES sources(id),
    created_at  timestamptz NOT NULL DEFAULT now(),
    CHECK ((pdf IS NULL) <> (pasted_text IS NULL))
);
CREATE INDEX uploads_subject_idx ON uploads (subject_id);

CREATE TABLE pages (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    upload_id   uuid NOT NULL REFERENCES uploads(id),
    number      int NOT NULL CHECK (number >= 1),
    raw_text    text NOT NULL,
    edited_text text NOT NULL,
    status      text NOT NULL DEFAULT 'needs_checking' CHECK (status IN ('needs_checking', 'ready')),
    image_png   bytea,
    updated_at  timestamptz NOT NULL DEFAULT now(),
    UNIQUE (upload_id, number)
);

-- Blocks made from a Ready page point back at it, so un-ready can remove them.
ALTER TABLE source_blocks ADD COLUMN page_id uuid REFERENCES pages(id);
CREATE INDEX source_blocks_page_idx ON source_blocks (page_id);

-- Claim checklist (D28, D20). Old claims keep nulls and stay valid.
-- Deleted claims are hidden, not removed: labels, cards and events point at them.
ALTER TABLE claims
    ADD COLUMN kind            text CHECK (kind IN ('definition', 'statement', 'condition', 'method')),
    ADD COLUMN author_user_id  uuid REFERENCES users(id),
    ADD COLUMN anchor_block_id uuid REFERENCES source_blocks(id),
    ADD COLUMN anchor_quote    text,
    ADD COLUMN created_at      timestamptz NOT NULL DEFAULT now(),
    ADD COLUMN deleted_at      timestamptz,
    ADD CONSTRAINT claims_anchor_pair CHECK ((anchor_block_id IS NULL) = (anchor_quote IS NULL));

-- Self-grades and disputes (D30). Neither touches review_events or any schedule.
CREATE TABLE self_grades (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id uuid NOT NULL REFERENCES teach_sessions(id),
    claim_id   uuid NOT NULL REFERENCES claims(id),
    user_id    uuid NOT NULL REFERENCES users(id),
    rating     text NOT NULL CHECK (rating IN ('right', 'not_sure', 'wrong')),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (session_id, claim_id)
);

CREATE TABLE disputes (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id uuid NOT NULL REFERENCES teach_sessions(id),
    -- Null when the disputed result had no matching claim.
    claim_id   uuid REFERENCES claims(id),
    user_id    uuid NOT NULL REFERENCES users(id),
    note       text NOT NULL DEFAULT '',
    status     text NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'reviewed')),
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX disputes_created_idx ON disputes (created_at);
