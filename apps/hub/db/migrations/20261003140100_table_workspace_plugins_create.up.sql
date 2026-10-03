BEGIN;

CREATE TABLE public.workspace_plugins
(
    workspace_id UUID        NOT NULL REFERENCES public.workspaces (id),
    plugin_id    TEXT        NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (workspace_id, plugin_id)
);

CREATE INDEX workspace_plugins_plugin_id_idx ON public.workspace_plugins (plugin_id);

CREATE TRIGGER set_updated_at
    BEFORE UPDATE
    ON public.workspace_plugins
    FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

COMMIT;
