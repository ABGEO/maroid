BEGIN;

CREATE TABLE customer_assignments
(
    id              UUID        NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    workspace_id    UUID        NOT NULL
        DEFAULT NULLIF(current_setting('app.workspace_id', true), '')::uuid
        REFERENCES public.workspaces (id),
    customer_number TEXT        NOT NULL CHECK (char_length(customer_number) BETWEEN 1 AND 64),
    assigned_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT customer_assignments_workspace_key UNIQUE (workspace_id)
);

COMMENT ON TABLE customer_assignments IS
    'Scoped to a workspace: the customer of the gas account whose transactions the workspace collects.';

ALTER TABLE customer_assignments
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE customer_assignments
    FORCE ROW LEVEL SECURITY;

CREATE POLICY customer_assignments_workspace_isolation ON customer_assignments
    USING (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid)
    WITH CHECK (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid);

COMMIT;
