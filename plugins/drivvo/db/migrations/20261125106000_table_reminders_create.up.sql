BEGIN;

CREATE TABLE reminders
(
    id                 UUID        NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    workspace_id       UUID        NOT NULL
        DEFAULT NULLIF(current_setting('app.workspace_id', true), '')::uuid
        REFERENCES public.workspaces (id) ON DELETE CASCADE,
    vehicle_id         UUID        NOT NULL REFERENCES vehicles (id) ON DELETE CASCADE,
    source_id          INTEGER     NOT NULL CHECK (source_id > 0),
    kind               TEXT        NOT NULL CHECK (kind IN ('maintenance', 'expense')),
    item_id            UUID        REFERENCES reference_items (id) ON DELETE SET NULL,
    due_on             DATE,
    due_odometer       NUMERIC,
    repeats            BOOLEAN     NOT NULL,
    repeat_every       INTEGER,
    repeat_unit        TEXT        CHECK (repeat_unit IN ('day', 'month', 'year')),
    repeat_distance    NUMERIC,
    note               TEXT,
    collected_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT reminders_source_key UNIQUE (workspace_id, source_id)
);

COMMENT ON TABLE reminders IS
    'Scoped to a workspace: the reminders of each vehicle.';

ALTER TABLE reminders
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE reminders
    FORCE ROW LEVEL SECURITY;

CREATE POLICY reminders_workspace_isolation ON reminders
    USING (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid)
    WITH CHECK (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid);

COMMIT;
