BEGIN;

CREATE TABLE IF NOT EXISTS plants
(
    id             TEXT        NOT NULL PRIMARY KEY,
    name           TEXT        NOT NULL,
    species        TEXT,
    environment_id TEXT        NOT NULL REFERENCES environments (id),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_plants_environment_id ON plants (environment_id);

CREATE TRIGGER set_updated_at
    BEFORE UPDATE ON plants
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

COMMIT;
