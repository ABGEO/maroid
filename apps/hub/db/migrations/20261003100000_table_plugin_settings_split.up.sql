BEGIN;

ALTER TABLE public.plugin_settings
    RENAME TO plugin_user_settings;
ALTER TABLE public.plugin_user_settings
    RENAME CONSTRAINT plugin_settings_pkey TO plugin_user_settings_pkey;
ALTER TABLE public.plugin_user_settings
    RENAME CONSTRAINT plugin_settings_user_plugin_key TO plugin_user_settings_user_plugin_key;
ALTER TABLE public.plugin_user_settings
    RENAME CONSTRAINT plugin_settings_user_id_fkey TO plugin_user_settings_user_id_fkey;
ALTER POLICY plugin_settings_user_isolation ON public.plugin_user_settings
    RENAME TO plugin_user_settings_user_isolation;

CREATE TABLE public.plugin_workspace_settings
(
    id           UUID        NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    workspace_id UUID        NOT NULL
        DEFAULT NULLIF(current_setting('app.workspace_id', true), '')::uuid
        REFERENCES public.workspaces (id),
    plugin_id    TEXT        NOT NULL,
    fields       JSONB       NOT NULL             DEFAULT '{}'::jsonb,
    created_at   TIMESTAMPTZ NOT NULL             DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL             DEFAULT NOW(),
    CONSTRAINT plugin_workspace_settings_workspace_plugin_key UNIQUE (workspace_id, plugin_id)
);

ALTER TABLE public.plugin_workspace_settings
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.plugin_workspace_settings
    FORCE ROW LEVEL SECURITY;

CREATE POLICY plugin_workspace_settings_workspace_isolation ON public.plugin_workspace_settings
    USING (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid)
    WITH CHECK (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid);

CREATE TRIGGER set_updated_at
    BEFORE UPDATE
    ON public.plugin_workspace_settings
    FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

COMMIT;
