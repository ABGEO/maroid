BEGIN;

-- The policy hides every row from a statement that acts in no workspace, and a
-- migration acts in none, so the owner reads the whole table until the end.
ALTER TABLE readings
    NO FORCE ROW LEVEL SECURITY;

DELETE
FROM readings older
    USING readings newer
WHERE older.workspace_id = newer.workspace_id
  AND older.customer_number = newer.customer_number
  AND older.serial_number = newer.serial_number
  AND older.reading_date = newer.reading_date
  AND (older.collected_at, older.id) < (newer.collected_at, newer.id);

ALTER TABLE readings
    DROP CONSTRAINT readings_workspace_hash_key,
    DROP COLUMN hash,
    DROP COLUMN previous_value;

DROP INDEX readings_customer_meter_date_idx;

ALTER TABLE readings
    ADD CONSTRAINT readings_workspace_meter_date_key
        UNIQUE (workspace_id, customer_number, serial_number, reading_date);

COMMENT ON TABLE readings IS
    'Scoped to a workspace: the value of each meter on each date. A reading outlives its assignment.';

ALTER TABLE readings
    FORCE ROW LEVEL SECURITY;

COMMIT;
