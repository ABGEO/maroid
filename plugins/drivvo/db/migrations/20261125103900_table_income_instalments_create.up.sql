BEGIN;

CREATE TABLE income_instalments
(
    id                 UUID        NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    workspace_id       UUID        NOT NULL
        DEFAULT NULLIF(current_setting('app.workspace_id', true), '')::uuid
        REFERENCES public.workspaces (id) ON DELETE CASCADE,
    income_id          UUID        NOT NULL REFERENCES incomes (id) ON DELETE CASCADE,
    number             SMALLINT    NOT NULL CHECK (number >= 1),
    due_on             DATE        NOT NULL,
    amount             NUMERIC     NOT NULL,
    collected_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT income_instalments_number_key UNIQUE (income_id, number)
);

CREATE INDEX income_instalments_income_idx
    ON income_instalments (workspace_id, income_id);

COMMENT ON TABLE income_instalments IS
    'Scoped to a workspace: the instalments of each income.';

ALTER TABLE income_instalments
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE income_instalments
    FORCE ROW LEVEL SECURITY;

CREATE POLICY income_instalments_workspace_isolation ON income_instalments
    USING (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid)
    WITH CHECK (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid);

COMMIT;
