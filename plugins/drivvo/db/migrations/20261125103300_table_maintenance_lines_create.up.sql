BEGIN;

CREATE TABLE maintenance_lines
(
    id                 UUID        NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    workspace_id       UUID        NOT NULL
        DEFAULT NULLIF(current_setting('app.workspace_id', true), '')::uuid
        REFERENCES public.workspaces (id) ON DELETE CASCADE,
    maintenance_id     UUID        NOT NULL REFERENCES maintenances (id) ON DELETE CASCADE,
    position           SMALLINT    NOT NULL CHECK (position >= 1),
    maintenance_type_id UUID        REFERENCES maintenance_types (id) ON DELETE SET NULL,
    amount             NUMERIC     NOT NULL,
    collected_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT maintenance_lines_position_key UNIQUE (maintenance_id, position)
);

CREATE INDEX maintenance_lines_maintenance_idx
    ON maintenance_lines (workspace_id, maintenance_id);

COMMENT ON TABLE maintenance_lines IS
    'Scoped to a workspace: each maintenance type of a maintenance.';

ALTER TABLE maintenance_lines
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE maintenance_lines
    FORCE ROW LEVEL SECURITY;

CREATE POLICY maintenance_lines_workspace_isolation ON maintenance_lines
    USING (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid)
    WITH CHECK (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid);

COMMIT;
