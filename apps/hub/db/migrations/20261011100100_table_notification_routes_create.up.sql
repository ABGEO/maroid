BEGIN;

CREATE TABLE public.notification_routes
(
    id           UUID        NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    workspace_id UUID        NOT NULL
        DEFAULT NULLIF(current_setting('app.workspace_id', true), '')::uuid
        REFERENCES public.workspaces (id),
    provider_id  UUID        NOT NULL,
    plugin_id    TEXT        NOT NULL,
    topic_id     TEXT        NOT NULL,
    destination  JSONB       NOT NULL             DEFAULT '{}'::jsonb,
    enabled      BOOLEAN     NOT NULL             DEFAULT TRUE,
    created_at   TIMESTAMPTZ NOT NULL             DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL             DEFAULT NOW(),
    -- NOTIF-INV-001, NOTIF-FR-009
    CONSTRAINT notification_routes_provider_fkey FOREIGN KEY (workspace_id, provider_id)
        REFERENCES public.notification_providers (workspace_id, id) ON DELETE CASCADE,
    -- NOTIF-FR-017
    CONSTRAINT notification_routes_unique_key
        UNIQUE (workspace_id, plugin_id, topic_id, provider_id, destination)
);

CREATE INDEX notification_routes_topic_idx
    ON public.notification_routes (workspace_id, plugin_id, topic_id);

ALTER TABLE public.notification_routes
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.notification_routes
    FORCE ROW LEVEL SECURITY;

-- NOTIF-INV-002
CREATE POLICY notification_routes_workspace_isolation ON public.notification_routes
    USING (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid)
    WITH CHECK (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid);

CREATE TRIGGER set_updated_at
    BEFORE UPDATE
    ON public.notification_routes
    FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

COMMIT;
