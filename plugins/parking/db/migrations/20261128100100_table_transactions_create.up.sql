BEGIN;

CREATE TABLE transactions
(
    id                  UUID           NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    user_id             UUID           NOT NULL
        DEFAULT NULLIF(current_setting('app.user_id', true), '')::uuid
        REFERENCES public.users (id),
    external_id         BIGINT         NOT NULL,
    kind                TEXT           NOT NULL
        CHECK (kind IN ('top_up', 'parking_payment', 'permit_purchase', 'other')),
    service_kind        TEXT           NOT NULL,
    amount              NUMERIC(12, 2) NOT NULL CHECK (amount >= 0),
    balance_after       NUMERIC(12, 2) NOT NULL,
    occurred_at         TIMESTAMPTZ    NOT NULL,
    session_external_id BIGINT,
    description         TEXT           NOT NULL,
    collected_at        TIMESTAMPTZ    NOT NULL DEFAULT now(),
    -- PARKHIST-FR-011: One transaction for one identifier of the parking service.
    CONSTRAINT transactions_user_external_id_key UNIQUE (user_id, external_id)
);

CREATE INDEX transactions_user_occurred_at_idx ON transactions (user_id, occurred_at DESC, id DESC);

COMMENT ON TABLE transactions IS
    'Scoped to a user: the top ups and the charges on the balance of the parking account of one person.';
COMMENT ON COLUMN transactions.session_external_id IS
    'The session that a parking payment pays. No foreign key: the payment can arrive before the session.';

ALTER TABLE transactions
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE transactions
    FORCE ROW LEVEL SECURITY;

-- PARKHIST-INV-001: A row of one user serves no other user, in any workspace.
CREATE POLICY transactions_user_isolation ON transactions
    USING (user_id = NULLIF(current_setting('app.user_id', true), '')::uuid)
    WITH CHECK (user_id = NULLIF(current_setting('app.user_id', true), '')::uuid);

COMMIT;
