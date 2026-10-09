BEGIN;

COMMENT ON TABLE transaction_types IS
    'Shared: the operation types of Tbilisi Energy, the same for every workspace.';

-- The rows predate the workspace and belong to none, so the table starts empty.
DROP TABLE transactions;

CREATE TABLE transactions
(
    id                   UUID        NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    workspace_id         UUID        NOT NULL
        DEFAULT NULLIF(current_setting('app.workspace_id', true), '')::uuid
        REFERENCES public.workspaces (id),
    hash                 TEXT        NOT NULL,
    transaction_type_id  INTEGER     NOT NULL REFERENCES transaction_types (id) ON DELETE RESTRICT,
    date                 TIMESTAMPTZ NOT NULL,
    consumption          DECIMAL     NOT NULL,
    amount               DECIMAL     NOT NULL,
    meter_reading        DECIMAL     NOT NULL,
    balance              DECIMAL     NOT NULL,
    billing_document_url TEXT        NOT NULL,
    meter_photo_url      TEXT        NOT NULL,
    CONSTRAINT transactions_workspace_hash_key UNIQUE (workspace_id, hash)
);

ALTER TABLE transactions
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE transactions
    FORCE ROW LEVEL SECURITY;

CREATE POLICY transactions_workspace_isolation ON transactions
    USING (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid)
    WITH CHECK (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid);

COMMIT;
