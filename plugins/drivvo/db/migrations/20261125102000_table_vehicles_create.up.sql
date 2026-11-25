BEGIN;

CREATE TABLE vehicles
(
    id                 UUID        NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    workspace_id       UUID        NOT NULL
        DEFAULT NULLIF(current_setting('app.workspace_id', true), '')::uuid
        REFERENCES public.workspaces (id) ON DELETE CASCADE,
    source_id          INTEGER     NOT NULL CHECK (source_id > 0),
    name               TEXT        NOT NULL,
    make               TEXT        NOT NULL,
    model              TEXT        NOT NULL,
    year               SMALLINT,
    distance_unit      TEXT        NOT NULL CHECK (distance_unit IN ('km', 'mi')),
    principal          BOOLEAN     NOT NULL,
    active             BOOLEAN     NOT NULL,
    collected_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT vehicles_source_key UNIQUE (workspace_id, source_id)
);

COMMENT ON TABLE vehicles IS
    'Scoped to a workspace: the vehicles of the car log account, an inactive one too.';

ALTER TABLE vehicles
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE vehicles
    FORCE ROW LEVEL SECURITY;

CREATE POLICY vehicles_workspace_isolation ON vehicles
    USING (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid)
    WITH CHECK (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid);

COMMIT;
