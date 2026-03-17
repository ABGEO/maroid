BEGIN;

CREATE TABLE IF NOT EXISTS environments
(
    id         TEXT        NOT NULL PRIMARY KEY,
    name       TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TRIGGER set_updated_at
    BEFORE UPDATE ON environments
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

COMMIT;
