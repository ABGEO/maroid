BEGIN;

CREATE TABLE transactions
(
    id               UUID        NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    workspace_id     UUID        NOT NULL
        DEFAULT NULLIF(current_setting('app.workspace_id', true), '')::uuid
        REFERENCES public.workspaces (id),
    customer_number  TEXT        NOT NULL CHECK (char_length(customer_number) BETWEEN 1 AND 64),
    operation        TEXT        NOT NULL,
    transaction_date DATE        NOT NULL,
    meter_value      NUMERIC,
    "usage"          NUMERIC,
    amount           NUMERIC,
    balance          NUMERIC,
    tariff           NUMERIC,
    meter_serial     TEXT,
    due_date         DATE,
    collected_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- The portal splits a reading in two charges on one date when the tariff changes,
    -- and answers no identifier of a transaction. A payment holds no meter value.
    CONSTRAINT transactions_identity_key
        UNIQUE NULLS NOT DISTINCT (workspace_id, customer_number, operation, transaction_date, meter_value)
);

COMMENT ON TABLE transactions IS
    'Scoped to a workspace: the transactions of a customer at the electricity service. A transaction outlives its assignment.';

ALTER TABLE transactions
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE transactions
    FORCE ROW LEVEL SECURITY;

CREATE POLICY transactions_workspace_isolation ON transactions
    USING (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid)
    WITH CHECK (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid);

COMMIT;
