BEGIN;

CREATE TABLE IF NOT EXISTS transactions
(
    hash                 TEXT PRIMARY KEY NOT NULL UNIQUE,
    transaction_type_id  INTEGER          NOT NULL REFERENCES transaction_types (id) ON DELETE RESTRICT,
    date                 TIMESTAMP        NOT NULL,
    consumption          DECIMAL          NOT NULL,
    amount               DECIMAL          NOT NULL,
    meter_reading        DECIMAL          NOT NULL,
    balance              DECIMAL          NOT NULL,
    billing_document_url TEXT             NOT NULL,
    meter_photo_url      TEXT             NOT NULL
);

COMMIT;
