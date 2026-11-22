BEGIN;

CREATE TABLE settings
(
    id           UUID        NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    user_id      UUID        NOT NULL
        DEFAULT NULLIF(current_setting('app.user_id', true), '')::uuid
        REFERENCES public.users (id),
    enabled      BOOLEAN     NOT NULL DEFAULT false,
    seeded_at    TIMESTAMPTZ,
    paused_until TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT settings_user_key UNIQUE (user_id)
);

COMMENT ON TABLE settings IS
    'Scoped to a user: the settings of one person, in every workspace.';

CREATE TRIGGER set_updated_at
    BEFORE UPDATE ON settings
    FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

ALTER TABLE settings
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE settings
    FORCE ROW LEVEL SECURITY;

CREATE POLICY settings_user_isolation ON settings
    USING (user_id = NULLIF(current_setting('app.user_id', true), '')::uuid)
    WITH CHECK (user_id = NULLIF(current_setting('app.user_id', true), '')::uuid);

COMMIT;
