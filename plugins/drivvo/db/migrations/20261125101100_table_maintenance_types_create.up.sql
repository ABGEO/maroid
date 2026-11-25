BEGIN;

CREATE TABLE maintenance_types
(
    id                 UUID        NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    workspace_id       UUID        NOT NULL
        DEFAULT NULLIF(current_setting('app.workspace_id', true), '')::uuid
        REFERENCES public.workspaces (id) ON DELETE CASCADE,
    source_id          INTEGER     NOT NULL CHECK (source_id > 0),
    name               TEXT        NOT NULL,
    collected_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT maintenance_types_source_key UNIQUE (workspace_id, source_id)
);

COMMENT ON TABLE maintenance_types IS
    'Scoped to a workspace: the maintenance types of the car log account.';

ALTER TABLE maintenance_types
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE maintenance_types
    FORCE ROW LEVEL SECURITY;

CREATE POLICY maintenance_types_workspace_isolation ON maintenance_types
    USING (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid)
    WITH CHECK (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid);

COMMIT;
