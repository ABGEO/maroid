BEGIN;

CREATE TABLE stations
(
    id                 UUID        NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    workspace_id       UUID        NOT NULL
        DEFAULT NULLIF(current_setting('app.workspace_id', true), '')::uuid
        REFERENCES public.workspaces (id) ON DELETE CASCADE,
    source_id          INTEGER     NOT NULL CHECK (source_id > 0),
    name               TEXT        NOT NULL,
    collected_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT stations_source_key UNIQUE (workspace_id, source_id)
);

COMMENT ON TABLE stations IS
    'Scoped to a workspace: the fuel stations of the car log account.';

ALTER TABLE stations
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE stations
    FORCE ROW LEVEL SECURITY;

CREATE POLICY stations_workspace_isolation ON stations
    USING (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid)
    WITH CHECK (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid);

COMMIT;
