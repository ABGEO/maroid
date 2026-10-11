BEGIN;

-- A user scopes the table, because the token of a personal account belongs to one
-- person in every workspace.
CREATE TABLE public.plugin_user_secrets
(
    id         UUID        NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    user_id    UUID        NOT NULL
        DEFAULT NULLIF(current_setting('app.user_id', true), '')::uuid
        REFERENCES public.users (id),
    plugin_id  TEXT        NOT NULL,
    name       TEXT        NOT NULL,
    ciphertext TEXT        NOT NULL,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL             DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL             DEFAULT NOW(),
    -- PSECRET-FR-012: The name holds 1 to 64 lowercase letters, digits, dots,
    -- underscores, and hyphens.
    CONSTRAINT plugin_user_secrets_name_check CHECK (name ~ '^[a-z0-9._-]{1,64}$'),
    -- PSECRET-FR-005: One value for each name of each plugin of each user, so a
    -- second write replaces it.
    CONSTRAINT plugin_user_secrets_user_plugin_name_key UNIQUE (user_id, plugin_id, name)
);

CREATE INDEX plugin_user_secrets_expires_at
    ON public.plugin_user_secrets (expires_at)
    WHERE expires_at IS NOT NULL;

ALTER TABLE public.plugin_user_secrets
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.plugin_user_secrets
    FORCE ROW LEVEL SECURITY;

-- PSECRET-INV-002: The plugin secret of one user reaches no other user.
CREATE POLICY plugin_user_secrets_user_isolation ON public.plugin_user_secrets
    USING (user_id = NULLIF(current_setting('app.user_id', true), '')::uuid)
    WITH CHECK (user_id = NULLIF(current_setting('app.user_id', true), '')::uuid);

CREATE TRIGGER set_updated_at
    BEFORE UPDATE
    ON public.plugin_user_secrets
    FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

COMMIT;
