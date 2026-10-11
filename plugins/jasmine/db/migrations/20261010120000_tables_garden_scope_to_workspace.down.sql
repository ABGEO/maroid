BEGIN;

DROP TABLE maintenance_entries;
DROP TABLE care_intervals;
DROP TABLE plants;
DROP TABLE environments;
DROP TABLE display_counters;

CREATE TABLE environments
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

CREATE TABLE plants
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
