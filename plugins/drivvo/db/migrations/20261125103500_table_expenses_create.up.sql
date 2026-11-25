BEGIN;

CREATE TABLE expenses
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
    title              TEXT,
    place_id           UUID        REFERENCES places (id) ON DELETE SET NULL,
    payment_method_id  UUID        REFERENCES payment_methods (id) ON DELETE SET NULL,
    reason_id          UUID        REFERENCES reasons (id) ON DELETE SET NULL,
    note               TEXT,
    collected_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT expenses_source_key UNIQUE (workspace_id, source_id)
);

CREATE INDEX expenses_list_idx
    ON expenses (workspace_id, vehicle_id, occurred_at DESC, id DESC);

COMMENT ON TABLE expenses IS
    'Scoped to a workspace: the expenses of each vehicle.';

ALTER TABLE expenses
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE expenses
    FORCE ROW LEVEL SECURITY;

CREATE POLICY expenses_workspace_isolation ON expenses
    USING (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid)
    WITH CHECK (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid);

COMMIT;
