BEGIN;

CREATE TABLE IF NOT EXISTS contributions
(
    hash              TEXT PRIMARY KEY NOT NULL UNIQUE,
    basis_id          UUID             NOT NULL,
    date              TIMESTAMP        NOT NULL,
    closing_date      TIMESTAMP        NULL,
    year              SMALLINT         NULL,
    month             SMALLINT         NULL,
    gross_salary      DECIMAL          NOT NULL,
    type              TEXT             NOT NULL,
    source            TEXT             NOT NULL,
    amount            DECIMAL          NULL,
    units             DECIMAL          NULL,
    organization_code TEXT             NULL REFERENCES organizations (code) ON DELETE RESTRICT
);

COMMIT;
