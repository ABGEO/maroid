BEGIN;

CREATE TABLE fuels
(
    id                 UUID        NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    workspace_id       UUID        NOT NULL
        DEFAULT NULLIF(current_setting('app.workspace_id', true), '')::uuid
        REFERENCES public.workspaces (id) ON DELETE CASCADE,
    source_id          INTEGER     NOT NULL CHECK (source_id > 0),
    name               TEXT        NOT NULL,
    collected_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT fuels_source_key UNIQUE (workspace_id, source_id)
);

COMMENT ON TABLE fuels IS
    'Scoped to a workspace: the fuels of the car log account.';

ALTER TABLE fuels
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE fuels
    FORCE ROW LEVEL SECURITY;

CREATE POLICY fuels_workspace_isolation ON fuels
    USING (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid)
    WITH CHECK (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid);

COMMIT;
