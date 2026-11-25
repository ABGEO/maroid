BEGIN;

CREATE TABLE log_entries
(
    id                 UUID        NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    workspace_id       UUID        NOT NULL
        DEFAULT NULLIF(current_setting('app.workspace_id', true), '')::uuid
        REFERENCES public.workspaces (id) ON DELETE CASCADE,
    vehicle_id         UUID        NOT NULL REFERENCES vehicles (id) ON DELETE CASCADE,
    kind               TEXT        NOT NULL CHECK (kind IN ('refuel', 'maintenance', 'expense', 'income',
                                                        'odometer')),
    source_id          INTEGER     NOT NULL CHECK (source_id > 0),
    occurred_at        TIMESTAMPTZ NOT NULL,
    odometer           NUMERIC     NOT NULL DEFAULT 0,
    amount             NUMERIC,
    title              TEXT,
    note               TEXT,
    station_id         UUID        REFERENCES reference_items (id) ON DELETE SET NULL,
    place_id           UUID        REFERENCES reference_items (id) ON DELETE SET NULL,
    payment_method_id  UUID        REFERENCES reference_items (id) ON DELETE SET NULL,
    reason_id          UUID        REFERENCES reference_items (id) ON DELETE SET NULL,
    income_type_id     UUID        REFERENCES reference_items (id) ON DELETE SET NULL,
    missed_refuel      BOOLEAN     NOT NULL DEFAULT false,
    battery_start_pct  NUMERIC,
    battery_end_pct    NUMERIC,
    charge_minutes     INTEGER,
    collected_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT log_entries_source_key UNIQUE (workspace_id, kind, source_id),
    CONSTRAINT log_entries_amount_check CHECK ((kind = 'odometer') = (amount IS NULL))
);

CREATE INDEX log_entries_list_idx
    ON log_entries (workspace_id, vehicle_id, occurred_at DESC, id DESC);

COMMENT ON TABLE log_entries IS
    'Scoped to a workspace: the refuels, the maintenances, the expenses, the incomes, and the odometer entries of each vehicle.';

ALTER TABLE log_entries
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE log_entries
    FORCE ROW LEVEL SECURITY;

CREATE POLICY log_entries_workspace_isolation ON log_entries
    USING (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid)
    WITH CHECK (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid);

COMMIT;
