BEGIN;

CREATE TABLE reference_items
(
    id                 UUID        NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    workspace_id       UUID        NOT NULL
        DEFAULT NULLIF(current_setting('app.workspace_id', true), '')::uuid
        REFERENCES public.workspaces (id) ON DELETE CASCADE,
    kind               TEXT        NOT NULL CHECK (kind IN ('fuel', 'maintenance_type', 'expense_type',
                                                        'income_type', 'reason', 'payment_method',
                                                        'station', 'place')),
    source_id          INTEGER     NOT NULL CHECK (source_id > 0),
    name               TEXT        NOT NULL,
    collected_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT reference_items_source_key UNIQUE (workspace_id, kind, source_id)
);

COMMENT ON TABLE reference_items IS
    'Scoped to a workspace: the fuels, the types, the stations, and the other named items of the car log account.';

ALTER TABLE reference_items
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE reference_items
    FORCE ROW LEVEL SECURITY;

CREATE POLICY reference_items_workspace_isolation ON reference_items
    USING (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid)
    WITH CHECK (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid);

COMMIT;
