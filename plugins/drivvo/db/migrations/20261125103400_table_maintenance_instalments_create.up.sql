BEGIN;

CREATE TABLE maintenance_instalments
(
    id                 UUID        NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    workspace_id       UUID        NOT NULL
        DEFAULT NULLIF(current_setting('app.workspace_id', true), '')::uuid
        REFERENCES public.workspaces (id) ON DELETE CASCADE,
    maintenance_id     UUID        NOT NULL REFERENCES maintenances (id) ON DELETE CASCADE,
    number             SMALLINT    NOT NULL CHECK (number >= 1),
    due_on             DATE        NOT NULL,
    amount             NUMERIC     NOT NULL,
    collected_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT maintenance_instalments_number_key UNIQUE (maintenance_id, number)
);

CREATE INDEX maintenance_instalments_maintenance_idx
    ON maintenance_instalments (workspace_id, maintenance_id);

COMMENT ON TABLE maintenance_instalments IS
    'Scoped to a workspace: the instalments of each maintenance.';

ALTER TABLE maintenance_instalments
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE maintenance_instalments
    FORCE ROW LEVEL SECURITY;

CREATE POLICY maintenance_instalments_workspace_isolation ON maintenance_instalments
    USING (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid)
    WITH CHECK (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid);

COMMIT;
