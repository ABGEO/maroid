BEGIN;

CREATE TABLE instalments
(
    id                 UUID        NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    workspace_id       UUID        NOT NULL
        DEFAULT NULLIF(current_setting('app.workspace_id', true), '')::uuid
        REFERENCES public.workspaces (id) ON DELETE CASCADE,
    log_entry_id       UUID        NOT NULL REFERENCES log_entries (id) ON DELETE CASCADE,
    number             SMALLINT    NOT NULL CHECK (number >= 1),
    due_on             DATE        NOT NULL,
    amount             NUMERIC     NOT NULL,
    collected_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT instalments_number_key UNIQUE (log_entry_id, number)
);

CREATE INDEX instalments_entry_idx
    ON instalments (workspace_id, log_entry_id);

COMMENT ON TABLE instalments IS
    'Scoped to a workspace: each instalment of a maintenance, an expense, or an income.';

ALTER TABLE instalments
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE instalments
    FORCE ROW LEVEL SECURITY;

CREATE POLICY instalments_workspace_isolation ON instalments
    USING (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid)
    WITH CHECK (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid);

COMMIT;
