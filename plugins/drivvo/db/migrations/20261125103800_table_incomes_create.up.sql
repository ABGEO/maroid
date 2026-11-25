BEGIN;

CREATE TABLE incomes
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
    income_type_id     UUID        REFERENCES income_types (id) ON DELETE SET NULL,
    note               TEXT,
    collected_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT incomes_source_key UNIQUE (workspace_id, source_id)
);

CREATE INDEX incomes_list_idx
    ON incomes (workspace_id, vehicle_id, occurred_at DESC, id DESC);

COMMENT ON TABLE incomes IS
    'Scoped to a workspace: the incomes of each vehicle.';

ALTER TABLE incomes
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE incomes
    FORCE ROW LEVEL SECURITY;

CREATE POLICY incomes_workspace_isolation ON incomes
    USING (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid)
    WITH CHECK (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid);

COMMIT;
