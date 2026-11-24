BEGIN;

CREATE TABLE subscriber_assignments
(
    id                UUID        NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    workspace_id      UUID        NOT NULL
        DEFAULT NULLIF(current_setting('app.workspace_id', true), '')::uuid
        REFERENCES public.workspaces (id) ON DELETE CASCADE,
    subscriber_number TEXT        NOT NULL CHECK (char_length(subscriber_number) BETWEEN 1 AND 64),
    kind              TEXT        NOT NULL CHECK (kind IN ('mobile', 'home_line')),
    assigned_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- MAGTIACC-INV-001: One workspace holds at most one assigned subscriber.
    CONSTRAINT subscriber_assignments_workspace_key UNIQUE (workspace_id)
);

COMMENT ON TABLE subscriber_assignments IS
    'Scoped to a workspace: the subscriber of the telecom account that the workspace follows.';

ALTER TABLE subscriber_assignments
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE subscriber_assignments
    FORCE ROW LEVEL SECURITY;

-- MAGTIACC-INV-002: An assigned subscriber of one workspace serves no other workspace.
CREATE POLICY subscriber_assignments_workspace_isolation ON subscriber_assignments
    USING (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid)
    WITH CHECK (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid);

COMMIT;
