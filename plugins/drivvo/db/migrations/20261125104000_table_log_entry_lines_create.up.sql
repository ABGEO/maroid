BEGIN;

CREATE TABLE log_entry_lines
(
    id                 UUID        NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    workspace_id       UUID        NOT NULL
        DEFAULT NULLIF(current_setting('app.workspace_id', true), '')::uuid
        REFERENCES public.workspaces (id) ON DELETE CASCADE,
    log_entry_id       UUID        NOT NULL REFERENCES log_entries (id) ON DELETE CASCADE,
    position           SMALLINT    NOT NULL CHECK (position >= 1),
    item_id            UUID        REFERENCES reference_items (id) ON DELETE SET NULL,
    amount             NUMERIC     NOT NULL,
    unit_price         NUMERIC,
    volume             NUMERIC,
    full_tank          BOOLEAN,
    free               BOOLEAN,
    collected_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT log_entry_lines_position_key UNIQUE (log_entry_id, position)
);

CREATE INDEX log_entry_lines_entry_idx
    ON log_entry_lines (workspace_id, log_entry_id);

COMMENT ON TABLE log_entry_lines IS
    'Scoped to a workspace: each fuel of a refuel, and each type of a maintenance or an expense.';

ALTER TABLE log_entry_lines
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE log_entry_lines
    FORCE ROW LEVEL SECURITY;

CREATE POLICY log_entry_lines_workspace_isolation ON log_entry_lines
    USING (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid)
    WITH CHECK (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid);

COMMIT;
