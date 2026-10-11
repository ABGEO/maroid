BEGIN;

CREATE TABLE public.notification_providers
(
    id           UUID        NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    workspace_id UUID        NOT NULL
        DEFAULT NULLIF(current_setting('app.workspace_id', true), '')::uuid
        REFERENCES public.workspaces (id),
    kind         TEXT        NOT NULL,
    name         TEXT        NOT NULL,
    fields       JSONB       NOT NULL             DEFAULT '{}'::jsonb,
    enabled      BOOLEAN     NOT NULL             DEFAULT TRUE,
    created_at   TIMESTAMPTZ NOT NULL             DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL             DEFAULT NOW(),
    CONSTRAINT notification_providers_name_length CHECK (char_length(name) BETWEEN 1 AND 64),
    -- NOTIF-INV-001: the composite key lets a route name a provider of its own workspace only.
    CONSTRAINT notification_providers_workspace_id_id_key UNIQUE (workspace_id, id)
);

ALTER TABLE public.notification_providers
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.notification_providers
    FORCE ROW LEVEL SECURITY;

-- NOTIF-INV-002
CREATE POLICY notification_providers_workspace_isolation ON public.notification_providers
    USING (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid)
    WITH CHECK (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid);

CREATE TRIGGER set_updated_at
    BEFORE UPDATE
    ON public.notification_providers
    FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

COMMIT;
