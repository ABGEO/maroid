BEGIN;

DROP TABLE IF EXISTS public.plugin_workspace_settings;

ALTER POLICY plugin_user_settings_user_isolation ON public.plugin_user_settings
    RENAME TO plugin_settings_user_isolation;
ALTER TABLE public.plugin_user_settings
    RENAME CONSTRAINT plugin_user_settings_user_id_fkey TO plugin_settings_user_id_fkey;
ALTER TABLE public.plugin_user_settings
    RENAME CONSTRAINT plugin_user_settings_user_plugin_key TO plugin_settings_user_plugin_key;
ALTER TABLE public.plugin_user_settings
    RENAME CONSTRAINT plugin_user_settings_pkey TO plugin_settings_pkey;
ALTER TABLE public.plugin_user_settings
    RENAME TO plugin_settings;

COMMIT;
