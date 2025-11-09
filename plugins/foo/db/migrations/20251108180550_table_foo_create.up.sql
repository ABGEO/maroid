BEGIN;

CREATE TABLE IF NOT EXISTS foo
(
    id         UUID PRIMARY KEY                     DEFAULT uuid_generate_v4(),
    bar        VARCHAR(9)                  NOT NULL UNIQUE,
    created_at TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
    updated_at TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT (now() AT TIME ZONE 'utc')
);

CREATE TRIGGER set_foo_updated_at
    BEFORE UPDATE
    ON foo
    FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

COMMIT;
