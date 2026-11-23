BEGIN;

CREATE TABLE rounds
(
    id               UUID        NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    user_id          UUID        NOT NULL
        DEFAULT NULLIF(current_setting('app.user_id', true), '')::uuid
        REFERENCES public.users (id),
    checkin_id       UUID,
    checkin_name     TEXT        NOT NULL,
    occurrence_at    TIMESTAMPTZ NOT NULL,
    state            TEXT        NOT NULL DEFAULT 'offered'
        CHECK (state IN ('offered', 'started', 'done', 'declined', 'expired')),
    start_message_id TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT rounds_user_id_key UNIQUE (user_id, id),
    -- PULSE-INV-003: One occurrence fires at most one round.
    CONSTRAINT rounds_occurrence_key UNIQUE (user_id, checkin_id, occurrence_at),
    CONSTRAINT rounds_checkin_fkey FOREIGN KEY (user_id, checkin_id)
        REFERENCES checkins (user_id, id) ON DELETE SET NULL (checkin_id)
);

CREATE INDEX rounds_open_idx ON rounds (user_id, occurrence_at)
    WHERE state IN ('offered', 'started');
CREATE UNIQUE INDEX rounds_start_message_idx ON rounds (user_id, start_message_id);

COMMENT ON TABLE rounds IS
    'Scoped to a user: the rounds that the check-ins of one person fire, in every workspace.';

CREATE TRIGGER set_updated_at
    BEFORE UPDATE ON rounds
    FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

ALTER TABLE rounds
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE rounds
    FORCE ROW LEVEL SECURITY;

CREATE POLICY rounds_user_isolation ON rounds
    USING (user_id = NULLIF(current_setting('app.user_id', true), '')::uuid)
    WITH CHECK (user_id = NULLIF(current_setting('app.user_id', true), '')::uuid);

COMMIT;
