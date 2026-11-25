BEGIN;

CREATE TABLE expense_lines
(
    id                 UUID        NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    workspace_id       UUID        NOT NULL
        DEFAULT NULLIF(current_setting('app.workspace_id', true), '')::uuid
        REFERENCES public.workspaces (id) ON DELETE CASCADE,
    expense_id         UUID        NOT NULL REFERENCES expenses (id) ON DELETE CASCADE,
    position           SMALLINT    NOT NULL CHECK (position >= 1),
    expense_type_id    UUID        REFERENCES expense_types (id) ON DELETE SET NULL,
    amount             NUMERIC     NOT NULL,
    collected_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT expense_lines_position_key UNIQUE (expense_id, position)
);

CREATE INDEX expense_lines_expense_idx
    ON expense_lines (workspace_id, expense_id);

COMMENT ON TABLE expense_lines IS
    'Scoped to a workspace: each expense type of an expense.';

ALTER TABLE expense_lines
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE expense_lines
    FORCE ROW LEVEL SECURITY;

CREATE POLICY expense_lines_workspace_isolation ON expense_lines
    USING (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid)
    WITH CHECK (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid);

COMMIT;
