BEGIN;

-- A workspace scopes the table, because the token of a shared account serves every
-- member of the workspace.
CREATE TABLE public.plugin_workspace_secrets
(
    id           UUID        NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    workspace_id UUID        NOT NULL
        DEFAULT NULLIF(current_setting('app.workspace_id', true), '')::uuid
        REFERENCES public.workspaces (id),
    plugin_id    TEXT        NOT NULL,
    name         TEXT        NOT NULL,
    ciphertext   TEXT        NOT NULL,
    expires_at   TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL             DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL             DEFAULT NOW(),
    -- PSECRET-FR-012: The name holds 1 to 64 lowercase letters, digits, dots,
    -- underscores, and hyphens.
    CONSTRAINT plugin_workspace_secrets_name_check CHECK (name ~ '^[a-z0-9._-]{1,64}$'),
    -- PSECRET-FR-005: One value for each name of each plugin of each workspace, so a
    -- second write replaces it.
    CONSTRAINT plugin_workspace_secrets_workspace_plugin_name_key UNIQUE (workspace_id, plugin_id, name)
);

CREATE INDEX plugin_workspace_secrets_expires_at
    ON public.plugin_workspace_secrets (expires_at)
    WHERE expires_at IS NOT NULL;

ALTER TABLE public.plugin_workspace_secrets
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.plugin_workspace_secrets
    FORCE ROW LEVEL SECURITY;

-- PSECRET-INV-002: The plugin secret of one workspace reaches no other workspace.
CREATE POLICY plugin_workspace_secrets_workspace_isolation ON public.plugin_workspace_secrets
    USING (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid)
    WITH CHECK (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid);

CREATE TRIGGER set_updated_at
    BEFORE UPDATE
    ON public.plugin_workspace_secrets
    FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

COMMIT;
