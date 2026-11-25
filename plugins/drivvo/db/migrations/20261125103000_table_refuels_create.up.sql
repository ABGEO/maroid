BEGIN;

CREATE TABLE refuels
(
    id                 UUID        NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    workspace_id       UUID        NOT NULL
        DEFAULT NULLIF(current_setting('app.workspace_id', true), '')::uuid
        REFERENCES public.workspaces (id) ON DELETE CASCADE,
    vehicle_id         UUID        NOT NULL REFERENCES vehicles (id) ON DELETE CASCADE,
    source_id          INTEGER     NOT NULL CHECK (source_id > 0),
    occurred_at        TIMESTAMPTZ NOT NULL,
    odometer           NUMERIC     NOT NULL DEFAULT 0,
    amount             NUMERIC     NOT NULL,
    station_id         UUID        REFERENCES stations (id) ON DELETE SET NULL,
    payment_method_id  UUID        REFERENCES payment_methods (id) ON DELETE SET NULL,
    reason_id          UUID        REFERENCES reasons (id) ON DELETE SET NULL,
    missed_refuel      BOOLEAN     NOT NULL,
    battery_start_pct  NUMERIC,
    battery_end_pct    NUMERIC,
    charge_minutes     INTEGER,
    note               TEXT,
    collected_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT refuels_source_key UNIQUE (workspace_id, source_id)
);

CREATE INDEX refuels_list_idx
    ON refuels (workspace_id, vehicle_id, occurred_at DESC, id DESC);

COMMENT ON TABLE refuels IS
    'Scoped to a workspace: the refuels and the charges of each vehicle.';

ALTER TABLE refuels
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE refuels
    FORCE ROW LEVEL SECURITY;

CREATE POLICY refuels_workspace_isolation ON refuels
    USING (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid)
    WITH CHECK (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid);

COMMIT;
