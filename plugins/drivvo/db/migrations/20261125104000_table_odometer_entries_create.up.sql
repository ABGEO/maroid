BEGIN;

CREATE TABLE odometer_entries
(
    id                 UUID        NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    workspace_id       UUID        NOT NULL
        DEFAULT NULLIF(current_setting('app.workspace_id', true), '')::uuid
        REFERENCES public.workspaces (id) ON DELETE CASCADE,
    vehicle_id         UUID        NOT NULL REFERENCES vehicles (id) ON DELETE CASCADE,
    source_id          INTEGER     NOT NULL CHECK (source_id > 0),
    occurred_at        TIMESTAMPTZ NOT NULL,
    odometer           NUMERIC     NOT NULL DEFAULT 0,
    note               TEXT,
    collected_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT odometer_entries_source_key UNIQUE (workspace_id, source_id)
);

CREATE INDEX odometer_entries_list_idx
    ON odometer_entries (workspace_id, vehicle_id, occurred_at DESC, id DESC);

COMMENT ON TABLE odometer_entries IS
    'Scoped to a workspace: the odometer entries of each vehicle.';

ALTER TABLE odometer_entries
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE odometer_entries
    FORCE ROW LEVEL SECURITY;

CREATE POLICY odometer_entries_workspace_isolation ON odometer_entries
    USING (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid)
    WITH CHECK (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid);

COMMIT;
