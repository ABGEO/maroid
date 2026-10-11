BEGIN;

-- The rows predate the workspace and belong to none, so the tables start empty.
DROP TABLE plants;
DROP TABLE environments;

CREATE TABLE display_counters
(
    id           UUID    NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    workspace_id UUID    NOT NULL
        DEFAULT NULLIF(current_setting('app.workspace_id', true), '')::uuid
        REFERENCES public.workspaces (id),
    kind         TEXT    NOT NULL CHECK (kind IN ('environment', 'plant')),
    last_number  INTEGER NOT NULL CHECK (last_number > 0),
    CONSTRAINT display_counters_workspace_kind_key UNIQUE (workspace_id, kind)
);

CREATE TABLE environments
(
    id           UUID        NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    workspace_id UUID        NOT NULL
        DEFAULT NULLIF(current_setting('app.workspace_id', true), '')::uuid
        REFERENCES public.workspaces (id),
    number       INTEGER     NOT NULL CHECK (number > 0),
    name         TEXT        NOT NULL CHECK (char_length(name) BETWEEN 1 AND 100),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT environments_workspace_number_key UNIQUE (workspace_id, number),
    CONSTRAINT environments_workspace_id_key UNIQUE (workspace_id, id)
);

CREATE TABLE plants
(
    id              UUID        NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    workspace_id    UUID        NOT NULL
        DEFAULT NULLIF(current_setting('app.workspace_id', true), '')::uuid
        REFERENCES public.workspaces (id),
    number          INTEGER     NOT NULL CHECK (number > 0),
    environment_id  UUID        NOT NULL,
    common_name     TEXT        NOT NULL CHECK (char_length(common_name) BETWEEN 1 AND 100),
    scientific_name TEXT CHECK (char_length(scientific_name) BETWEEN 1 AND 150),
    acquired_on     DATE,
    care_notes      TEXT CHECK (char_length(care_notes) BETWEEN 1 AND 10000),
    status          TEXT        NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive')),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT plants_workspace_number_key UNIQUE (workspace_id, number),
    CONSTRAINT plants_workspace_id_key UNIQUE (workspace_id, id),
    CONSTRAINT plants_environment_fkey FOREIGN KEY (workspace_id, environment_id)
        REFERENCES environments (workspace_id, id) ON DELETE RESTRICT
);

CREATE INDEX plants_workspace_environment_idx ON plants (workspace_id, environment_id);

CREATE TABLE care_intervals
(
    id           UUID        NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    workspace_id UUID        NOT NULL
        DEFAULT NULLIF(current_setting('app.workspace_id', true), '')::uuid
        REFERENCES public.workspaces (id),
    plant_id     UUID        NOT NULL,
    type         TEXT        NOT NULL CHECK (type IN ('watering', 'fertilizing', 'pruning', 'repotting',
                                                      'pest_treatment', 'cleaning', 'misting', 'other')),
    days         SMALLINT    NOT NULL CHECK (days BETWEEN 1 AND 365),
    set_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT care_intervals_workspace_plant_type_key UNIQUE (workspace_id, plant_id, type),
    CONSTRAINT care_intervals_plant_fkey FOREIGN KEY (workspace_id, plant_id)
        REFERENCES plants (workspace_id, id) ON DELETE CASCADE
);

CREATE TABLE maintenance_entries
(
    id           UUID        NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    workspace_id UUID        NOT NULL
        DEFAULT NULLIF(current_setting('app.workspace_id', true), '')::uuid
        REFERENCES public.workspaces (id),
    plant_id     UUID        NOT NULL,
    type         TEXT        NOT NULL CHECK (type IN ('watering', 'fertilizing', 'pruning', 'repotting',
                                                      'pest_treatment', 'cleaning', 'misting', 'other')),
    performed_at TIMESTAMPTZ NOT NULL,
    note         TEXT CHECK (char_length(note) BETWEEN 1 AND 1000),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT maintenance_entries_plant_fkey FOREIGN KEY (workspace_id, plant_id)
        REFERENCES plants (workspace_id, id) ON DELETE CASCADE
);

CREATE INDEX maintenance_entries_latest_idx
    ON maintenance_entries (workspace_id, plant_id, type, performed_at DESC);
CREATE INDEX maintenance_entries_log_idx
    ON maintenance_entries (workspace_id, performed_at DESC, id DESC);

CREATE TRIGGER set_updated_at
    BEFORE UPDATE ON environments
    FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER set_updated_at
    BEFORE UPDATE ON plants
    FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER set_updated_at
    BEFORE UPDATE ON care_intervals
    FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER set_updated_at
    BEFORE UPDATE ON maintenance_entries
    FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

ALTER TABLE display_counters
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE display_counters
    FORCE ROW LEVEL SECURITY;
CREATE POLICY display_counters_workspace_isolation ON display_counters
    USING (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid)
    WITH CHECK (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid);

ALTER TABLE environments
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE environments
    FORCE ROW LEVEL SECURITY;
CREATE POLICY environments_workspace_isolation ON environments
    USING (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid)
    WITH CHECK (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid);

ALTER TABLE plants
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE plants
    FORCE ROW LEVEL SECURITY;
CREATE POLICY plants_workspace_isolation ON plants
    USING (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid)
    WITH CHECK (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid);

ALTER TABLE care_intervals
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE care_intervals
    FORCE ROW LEVEL SECURITY;
CREATE POLICY care_intervals_workspace_isolation ON care_intervals
    USING (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid)
    WITH CHECK (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid);

ALTER TABLE maintenance_entries
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE maintenance_entries
    FORCE ROW LEVEL SECURITY;
CREATE POLICY maintenance_entries_workspace_isolation ON maintenance_entries
    USING (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid)
    WITH CHECK (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid);

COMMIT;
