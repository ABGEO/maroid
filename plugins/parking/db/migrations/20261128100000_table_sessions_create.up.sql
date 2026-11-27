BEGIN;

CREATE TABLE sessions
(
    id           UUID           NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    user_id      UUID           NOT NULL
        DEFAULT NULLIF(current_setting('app.user_id', true), '')::uuid
        REFERENCES public.users (id),
    external_id  BIGINT         NOT NULL,
    lot_number   TEXT           NOT NULL CHECK (char_length(lot_number) BETWEEN 1 AND 32),
    address      TEXT           NOT NULL,
    plate        TEXT           NOT NULL CHECK (char_length(plate) BETWEEN 1 AND 16),
    parking_type TEXT           NOT NULL,
    started_at   TIMESTAMPTZ    NOT NULL,
    ended_at     TIMESTAMPTZ    NOT NULL,
    amount       NUMERIC(12, 2) NOT NULL CHECK (amount >= 0),
    collected_at TIMESTAMPTZ    NOT NULL DEFAULT now(),
    CONSTRAINT sessions_ended_after_start CHECK (ended_at >= started_at),
    -- PARKHIST-FR-011: One session for one identifier of the parking service.
    CONSTRAINT sessions_user_external_id_key UNIQUE (user_id, external_id)
);

CREATE INDEX sessions_user_started_at_idx ON sessions (user_id, started_at DESC, id DESC);

COMMENT ON TABLE sessions IS
    'Scoped to a user: the finished parking sessions of the parking account of one person.';

ALTER TABLE sessions
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE sessions
    FORCE ROW LEVEL SECURITY;

-- PARKHIST-INV-001: A row of one user serves no other user, in any workspace.
CREATE POLICY sessions_user_isolation ON sessions
    USING (user_id = NULLIF(current_setting('app.user_id', true), '')::uuid)
    WITH CHECK (user_id = NULLIF(current_setting('app.user_id', true), '')::uuid);

COMMIT;
