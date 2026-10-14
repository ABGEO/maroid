BEGIN;

CREATE TABLE bills
(
    id              UUID        NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    workspace_id    UUID        NOT NULL
        DEFAULT NULLIF(current_setting('app.workspace_id', true), '')::uuid
        REFERENCES public.workspaces (id),
    customer_number TEXT        NOT NULL CHECK (char_length(customer_number) BETWEEN 1 AND 64),
    bill_number     TEXT        NOT NULL,
    service_bill_id UUID        NOT NULL,
    month           DATE        NOT NULL CHECK (extract(DAY FROM month) = 1),
    issued_at       TIMESTAMPTZ NOT NULL,
    meter_serial    TEXT,
    collected_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT bills_workspace_bill_number_key UNIQUE (workspace_id, bill_number)
);

COMMENT ON TABLE bills IS
    'Scoped to a workspace: the bills of a customer at the electricity service. The document stays at the service.';

ALTER TABLE bills
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE bills
    FORCE ROW LEVEL SECURITY;

-- ELECACC-INV-002: A bill of one workspace serves no other workspace.
CREATE POLICY bills_workspace_isolation ON bills
    USING (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid)
    WITH CHECK (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid);

COMMIT;
