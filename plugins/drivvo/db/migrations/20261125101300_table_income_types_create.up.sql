BEGIN;

CREATE TABLE income_types
(
    id                 UUID        NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    workspace_id       UUID        NOT NULL
        DEFAULT NULLIF(current_setting('app.workspace_id', true), '')::uuid
        REFERENCES public.workspaces (id) ON DELETE CASCADE,
    source_id          INTEGER     NOT NULL CHECK (source_id > 0),
    name               TEXT        NOT NULL,
    collected_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT income_types_source_key UNIQUE (workspace_id, source_id)
);

COMMENT ON TABLE income_types IS
    'Scoped to a workspace: the income types of the car log account.';

ALTER TABLE income_types
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE income_types
    FORCE ROW LEVEL SECURITY;

CREATE POLICY income_types_workspace_isolation ON income_types
    USING (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid)
    WITH CHECK (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid);

COMMIT;
