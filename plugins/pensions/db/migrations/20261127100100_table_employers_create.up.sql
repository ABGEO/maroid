BEGIN;

CREATE TABLE employers
(
    id           UUID        NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    code         TEXT        NOT NULL CHECK (char_length(code) BETWEEN 1 AND 64),
    name         TEXT        NOT NULL,
    collected_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT employers_code_key UNIQUE (code)
);

COMMENT ON TABLE employers IS
    'Shared: the employers that the pension service names. An employer names no person, and two people of one employer read the same row.';

COMMIT;
