BEGIN;

-- A stored row names no customer, so the collection and the backfill fill the table again.
DROP TABLE transactions;

CREATE TABLE transactions
(
    id                  UUID        NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    workspace_id        UUID        NOT NULL
        DEFAULT NULLIF(current_setting('app.workspace_id', true), '')::uuid
        REFERENCES public.workspaces (id),
    customer_number     TEXT        NOT NULL CHECK (char_length(customer_number) BETWEEN 1 AND 64),
    transaction_type_id INTEGER     NOT NULL REFERENCES transaction_types (id) ON DELETE RESTRICT,
    transaction_date    DATE        NOT NULL,
    consumption         NUMERIC     NOT NULL,
    meter_value         NUMERIC     NOT NULL,
    amount              NUMERIC     NOT NULL,
    balance             NUMERIC     NOT NULL,
    bill_path           TEXT,
    meter_photo_path    TEXT,
    collected_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT transactions_workspace_customer_operation_date_key
        UNIQUE (workspace_id, customer_number, transaction_type_id, transaction_date)
);

COMMENT ON TABLE transactions IS
    'Scoped to a workspace: the transactions of a customer at the gas service. A transaction outlives its assignment.';

ALTER TABLE transactions
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE transactions
    FORCE ROW LEVEL SECURITY;

CREATE POLICY transactions_workspace_isolation ON transactions
    USING (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid)
    WITH CHECK (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid);

COMMIT;
