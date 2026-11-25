BEGIN;

CREATE TABLE reasons
(
    id                 UUID        NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    workspace_id       UUID        NOT NULL
        DEFAULT NULLIF(current_setting('app.workspace_id', true), '')::uuid
        REFERENCES public.workspaces (id) ON DELETE CASCADE,
    source_id          INTEGER     NOT NULL CHECK (source_id > 0),
    name               TEXT        NOT NULL,
    collected_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT reasons_source_key UNIQUE (workspace_id, source_id)
);

COMMENT ON TABLE reasons IS
    'Scoped to a workspace: the reasons of a refuel or an expense.';

ALTER TABLE reasons
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE reasons
    FORCE ROW LEVEL SECURITY;

CREATE POLICY reasons_workspace_isolation ON reasons
    USING (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid)
    WITH CHECK (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid);

COMMIT;
