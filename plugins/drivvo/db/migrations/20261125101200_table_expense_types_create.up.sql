BEGIN;

CREATE TABLE expense_types
(
    id                 UUID        NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    workspace_id       UUID        NOT NULL
        DEFAULT NULLIF(current_setting('app.workspace_id', true), '')::uuid
        REFERENCES public.workspaces (id) ON DELETE CASCADE,
    source_id          INTEGER     NOT NULL CHECK (source_id > 0),
    name               TEXT        NOT NULL,
    collected_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT expense_types_source_key UNIQUE (workspace_id, source_id)
);

COMMENT ON TABLE expense_types IS
    'Scoped to a workspace: the expense types of the car log account.';

ALTER TABLE expense_types
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE expense_types
    FORCE ROW LEVEL SECURITY;

CREATE POLICY expense_types_workspace_isolation ON expense_types
    USING (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid)
    WITH CHECK (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid);

COMMIT;
