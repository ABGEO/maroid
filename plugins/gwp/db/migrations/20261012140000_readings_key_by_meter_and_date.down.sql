BEGIN;

ALTER TABLE readings
    NO FORCE ROW LEVEL SECURITY;

ALTER TABLE readings
    DROP CONSTRAINT readings_workspace_meter_date_key,
    ADD COLUMN previous_value NUMERIC NOT NULL DEFAULT 0,
    ADD COLUMN hash TEXT;

-- The values that the hash covered are gone, so the identifier of each row stands in.
UPDATE readings
SET hash = id::text;

ALTER TABLE readings
    ALTER COLUMN previous_value DROP DEFAULT,
    ALTER COLUMN hash SET NOT NULL,
    ADD CONSTRAINT readings_workspace_hash_key UNIQUE (workspace_id, hash);

CREATE INDEX readings_customer_meter_date_idx
    ON readings (workspace_id, customer_number, serial_number, reading_date DESC);

COMMENT ON TABLE readings IS
    'Scoped to a workspace: the meter readings that the workspace collected. A reading outlives its assignment.';

ALTER TABLE readings
    FORCE ROW LEVEL SECURITY;

COMMIT;
