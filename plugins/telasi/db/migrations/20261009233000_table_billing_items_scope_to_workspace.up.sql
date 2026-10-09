BEGIN;

-- The rows predate the workspace and belong to none, so the table starts empty.
DROP TABLE billing_items;

CREATE TABLE billing_items
(
    id           UUID        NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    workspace_id UUID        NOT NULL
        DEFAULT NULLIF(current_setting('app.workspace_id', true), '')::uuid
        REFERENCES public.workspaces (id),
    hash         TEXT        NOT NULL,
    operation    TEXT        NOT NULL,
    reading      DECIMAL     NOT NULL,
    consumption  DECIMAL     NOT NULL,
    amount       DECIMAL     NOT NULL,
    date         TIMESTAMPTZ NOT NULL,
    CONSTRAINT billing_items_workspace_hash_key UNIQUE (workspace_id, hash)
);

ALTER TABLE billing_items
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE billing_items
    FORCE ROW LEVEL SECURITY;

CREATE POLICY billing_items_workspace_isolation ON billing_items
    USING (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid)
    WITH CHECK (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid);

COMMIT;
