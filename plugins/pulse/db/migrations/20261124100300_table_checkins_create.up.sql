BEGIN;

CREATE TABLE checkins
(
    id          UUID        NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    user_id     UUID        NOT NULL
        DEFAULT NULLIF(current_setting('app.user_id', true), '')::uuid
        REFERENCES public.users (id),
    name        TEXT        NOT NULL CHECK (char_length(name) BETWEEN 1 AND 40),
    local_times TIME[]      NOT NULL CHECK (cardinality(local_times) BETWEEN 1 AND 6),
    -- ISO weekdays, 1 for Monday. An empty array is every day.
    weekdays    SMALLINT[]  NOT NULL DEFAULT '{}' CHECK (weekdays <@ '{1,2,3,4,5,6,7}'),
    enabled     BOOLEAN     NOT NULL DEFAULT true,
    fires_after TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT checkins_user_id_key UNIQUE (user_id, id)
);

COMMENT ON TABLE checkins IS
    'Scoped to a user: the check-ins of one person, in every workspace.';

CREATE TRIGGER set_updated_at
    BEFORE UPDATE ON checkins
    FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

ALTER TABLE checkins
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE checkins
    FORCE ROW LEVEL SECURITY;

CREATE POLICY checkins_user_isolation ON checkins
    USING (user_id = NULLIF(current_setting('app.user_id', true), '')::uuid)
    WITH CHECK (user_id = NULLIF(current_setting('app.user_id', true), '')::uuid);

COMMIT;
