BEGIN;

CREATE TABLE expense_instalments
(
    id                 UUID        NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    workspace_id       UUID        NOT NULL
        DEFAULT NULLIF(current_setting('app.workspace_id', true), '')::uuid
        REFERENCES public.workspaces (id) ON DELETE CASCADE,
    expense_id         UUID        NOT NULL REFERENCES expenses (id) ON DELETE CASCADE,
    number             SMALLINT    NOT NULL CHECK (number >= 1),
    due_on             DATE        NOT NULL,
    amount             NUMERIC     NOT NULL,
    collected_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT expense_instalments_number_key UNIQUE (expense_id, number)
);

CREATE INDEX expense_instalments_expense_idx
    ON expense_instalments (workspace_id, expense_id);

COMMENT ON TABLE expense_instalments IS
    'Scoped to a workspace: the instalments of each expense.';

ALTER TABLE expense_instalments
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE expense_instalments
    FORCE ROW LEVEL SECURITY;

CREATE POLICY expense_instalments_workspace_isolation ON expense_instalments
    USING (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid)
    WITH CHECK (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid);

COMMIT;
