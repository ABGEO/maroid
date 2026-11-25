BEGIN;

CREATE TABLE refuel_fuels
(
    id                 UUID        NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    workspace_id       UUID        NOT NULL
        DEFAULT NULLIF(current_setting('app.workspace_id', true), '')::uuid
        REFERENCES public.workspaces (id) ON DELETE CASCADE,
    refuel_id          UUID        NOT NULL REFERENCES refuels (id) ON DELETE CASCADE,
    position           SMALLINT    NOT NULL CHECK (position BETWEEN 1 AND 3),
    fuel_id            UUID        REFERENCES fuels (id) ON DELETE SET NULL,
    amount             NUMERIC     NOT NULL,
    unit_price         NUMERIC     NOT NULL,
    volume             NUMERIC     NOT NULL,
    full_tank          BOOLEAN     NOT NULL,
    free               BOOLEAN     NOT NULL,
    collected_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT refuel_fuels_position_key UNIQUE (refuel_id, position)
);

CREATE INDEX refuel_fuels_refuel_idx
    ON refuel_fuels (workspace_id, refuel_id);

COMMENT ON TABLE refuel_fuels IS
    'Scoped to a workspace: each fuel of a refuel.';

ALTER TABLE refuel_fuels
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE refuel_fuels
    FORCE ROW LEVEL SECURITY;

CREATE POLICY refuel_fuels_workspace_isolation ON refuel_fuels
    USING (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid)
    WITH CHECK (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid);

COMMIT;
