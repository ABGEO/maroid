BEGIN;

CREATE TABLE places
(
    id                 UUID        NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    workspace_id       UUID        NOT NULL
        DEFAULT NULLIF(current_setting('app.workspace_id', true), '')::uuid
        REFERENCES public.workspaces (id) ON DELETE CASCADE,
    source_id          INTEGER     NOT NULL CHECK (source_id > 0),
    name               TEXT        NOT NULL,
    collected_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT places_source_key UNIQUE (workspace_id, source_id)
);

COMMENT ON TABLE places IS
    'Scoped to a workspace: the places of a maintenance or an expense.';

ALTER TABLE places
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE places
    FORCE ROW LEVEL SECURITY;

CREATE POLICY places_workspace_isolation ON places
    USING (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid)
    WITH CHECK (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid);

COMMIT;
