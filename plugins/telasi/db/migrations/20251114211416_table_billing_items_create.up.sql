BEGIN;

CREATE TABLE IF NOT EXISTS billing_items
(
    hash        TEXT PRIMARY KEY NOT NULL UNIQUE,
    operation   TEXT             NOT NULL,
    reading     DECIMAL          NOT NULL,
    consumption DECIMAL          NOT NULL,
    amount      DECIMAL          NOT NULL,
    date        TIMESTAMP        NOT NULL
);

COMMIT;
