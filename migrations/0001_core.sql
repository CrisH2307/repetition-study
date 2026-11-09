-- Core schema. IDs are uuids. Up only: fix forward with a new file.

CREATE TABLE users (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name       text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE subjects (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      uuid NOT NULL REFERENCES users(id) DEFAULT '00000000-0000-0000-0000-000000000001',
    name         text NOT NULL,
    weekly_hours numeric NOT NULL CHECK (weekly_hours >= 0),
    level        text NOT NULL CHECK (level IN ('beginner', 'intermediate', 'advanced')),
    exam_date    date,
    weight       numeric NOT NULL DEFAULT 1 CHECK (weight >= 0),
    created_at   timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE sources (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    kind        text NOT NULL CHECK (kind IN ('course', 'allowlist')),
    title       text NOT NULL,
    url         text NOT NULL DEFAULT '',
    publisher   text NOT NULL DEFAULT '',
    license     text NOT NULL DEFAULT '',
    reviewed_by text NOT NULL DEFAULT '',
    reviewed_at timestamptz,
    text        text NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE concepts (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    subject_id uuid NOT NULL REFERENCES subjects(id),
    name       text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (subject_id, name)
);

CREATE TABLE claims (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    concept_id uuid NOT NULL REFERENCES concepts(id),
    text       text NOT NULL,
    source_id  uuid NOT NULL REFERENCES sources(id),
    locator    text NOT NULL DEFAULT '',
    required   boolean NOT NULL DEFAULT true
);

CREATE TABLE teach_sessions (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    concept_id  uuid NOT NULL REFERENCES concepts(id),
    status      text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'finished')),
    started_at  timestamptz NOT NULL DEFAULT now(),
    finished_at timestamptz
);

CREATE TABLE teach_turns (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id     uuid NOT NULL REFERENCES teach_sessions(id),
    turn_no        int NOT NULL,
    transcript     text NOT NULL,
    latex          text NOT NULL DEFAULT '',
    assistant_text text NOT NULL DEFAULT '',
    created_at     timestamptz NOT NULL DEFAULT now(),
    UNIQUE (session_id, turn_no)
);

CREATE TABLE taught_claims (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id   uuid NOT NULL REFERENCES teach_sessions(id),
    concept_id   uuid NOT NULL REFERENCES concepts(id),
    student_text text NOT NULL,
    turn_no      int NOT NULL
);

CREATE TABLE claim_labels (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id        uuid NOT NULL REFERENCES teach_sessions(id),
    claim_id          uuid REFERENCES claims(id),
    student_text      text NOT NULL,
    label             text NOT NULL CHECK (label IN ('supported', 'partial', 'contradicted', 'absent')),
    citation_source_id uuid REFERENCES sources(id),
    citation_quote    text,
    citation_locator  text,
    correction        text,
    probe_hint        text,
    created_at        timestamptz NOT NULL DEFAULT now(),
    -- A correction is only allowed with a citation. Code also verifies the quote.
    CHECK (correction IS NULL OR (citation_source_id IS NOT NULL AND citation_quote IS NOT NULL))
);

CREATE TABLE cards (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    concept_id uuid NOT NULL REFERENCES concepts(id),
    claim_id   uuid REFERENCES claims(id),
    front      text NOT NULL,
    back       text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- One FSRS state per concept. Cards carry no schedule of their own.
CREATE TABLE concept_schedules (
    concept_id     uuid PRIMARY KEY REFERENCES concepts(id),
    due_at         timestamptz NOT NULL,
    stability      double precision NOT NULL DEFAULT 0,
    difficulty     double precision NOT NULL DEFAULT 0,
    state          int NOT NULL DEFAULT 0,
    reps           int NOT NULL DEFAULT 0,
    lapses         int NOT NULL DEFAULT 0,
    last_review_at timestamptz
);

CREATE TABLE review_events (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type       text NOT NULL CHECK (event_type IN ('card_review', 'teach_back', 'manual_override')),
    concept_id       uuid NOT NULL REFERENCES concepts(id),
    card_id          uuid REFERENCES cards(id),
    grade            text CHECK (grade IN ('again', 'hard', 'good', 'easy')),
    due_before       timestamptz NOT NULL,
    due_after        timestamptz NOT NULL,
    stability_before double precision NOT NULL,
    stability_after  double precision NOT NULL,
    payload          jsonb NOT NULL DEFAULT '{}',
    created_at       timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX review_events_concept_idx ON review_events (concept_id, created_at);

CREATE TABLE plan_blocks (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    subject_id uuid NOT NULL REFERENCES subjects(id),
    concept_id uuid REFERENCES concepts(id),
    kind       text NOT NULL CHECK (kind IN ('review', 'teach', 'new')),
    starts_at  timestamptz NOT NULL,
    minutes    int NOT NULL CHECK (minutes > 0),
    reason     text NOT NULL DEFAULT '',
    locked     boolean NOT NULL DEFAULT false
);
CREATE INDEX plan_blocks_starts_idx ON plan_blocks (starts_at);

CREATE TABLE digests (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     uuid NOT NULL REFERENCES users(id),
    digest_date date NOT NULL,
    payload     jsonb NOT NULL DEFAULT '{}',
    created_at  timestamptz NOT NULL DEFAULT now(),
    UNIQUE (user_id, digest_date)
);
