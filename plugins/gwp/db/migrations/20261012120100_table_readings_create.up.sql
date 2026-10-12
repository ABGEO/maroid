BEGIN;

CREATE TABLE readings
(
    id              UUID        NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    workspace_id    UUID        NOT NULL
        DEFAULT NULLIF(current_setting('app.workspace_id', true), '')::uuid
        REFERENCES public.workspaces (id),
    hash            TEXT        NOT NULL,
    customer_number TEXT        NOT NULL,
    serial_number   TEXT        NOT NULL,
    reading_date    DATE        NOT NULL,
    value           NUMERIC     NOT NULL,
    previous_value  NUMERIC     NOT NULL,
    collected_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT readings_workspace_hash_key UNIQUE (workspace_id, hash)
);

COMMENT ON TABLE readings IS
    'Scoped to a workspace: the meter readings that the workspace collected. A reading outlives its assignment.';

CREATE INDEX readings_customer_meter_date_idx
    ON readings (workspace_id, customer_number, serial_number, reading_date DESC);

ALTER TABLE readings
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE readings
    FORCE ROW LEVEL SECURITY;

CREATE POLICY readings_workspace_isolation ON readings
    USING (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid)
    WITH CHECK (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid);

COMMIT;
