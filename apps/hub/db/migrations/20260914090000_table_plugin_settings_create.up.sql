BEGIN;

CREATE TABLE public.plugin_settings
(
    id         UUID        NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    user_id    UUID        NOT NULL
        DEFAULT NULLIF(current_setting('app.user_id', true), '')::uuid
        REFERENCES public.users (id),
    plugin_id  TEXT        NOT NULL,
    fields     JSONB       NOT NULL             DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL             DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL             DEFAULT NOW(),
    CONSTRAINT plugin_settings_user_plugin_key UNIQUE (user_id, plugin_id)
);

ALTER TABLE public.plugin_settings
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.plugin_settings
    FORCE ROW LEVEL SECURITY;

CREATE POLICY plugin_settings_user_isolation ON public.plugin_settings
    USING (user_id = NULLIF(current_setting('app.user_id', true), '')::uuid)
    WITH CHECK (user_id = NULLIF(current_setting('app.user_id', true), '')::uuid);

CREATE TRIGGER set_updated_at
    BEFORE UPDATE
    ON public.plugin_settings
    FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

COMMIT;
