BEGIN;

CREATE TABLE moments
(
    id           UUID        NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    workspace_id UUID        NOT NULL
        DEFAULT NULLIF(current_setting('app.workspace_id', true), '')::uuid
        REFERENCES public.workspaces (id),
    plant_id     UUID        NOT NULL,
    taken_on     DATE        NOT NULL,
    note         TEXT CHECK (char_length(note) BETWEEN 1 AND 1000),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT moments_workspace_plant_id_key UNIQUE (workspace_id, plant_id, id),
    -- GARDEN-INV-007: A moment and its plant are in one workspace.
    CONSTRAINT moments_plant_fkey FOREIGN KEY (workspace_id, plant_id)
        REFERENCES plants (workspace_id, id) ON DELETE CASCADE
);

CREATE INDEX moments_gallery_idx ON moments (workspace_id, plant_id, taken_on DESC, id DESC);

CREATE TABLE photos
(
    id           UUID        NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    workspace_id UUID        NOT NULL
        DEFAULT NULLIF(current_setting('app.workspace_id', true), '')::uuid
        REFERENCES public.workspaces (id),
    plant_id     UUID        NOT NULL,
    moment_id    UUID        NOT NULL,
    file_id      UUID        NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT photos_workspace_plant_id_key UNIQUE (workspace_id, plant_id, id),
    -- GARDEN-INV-007: A photo, its moment, and its plant are in one workspace.
    CONSTRAINT photos_moment_fkey FOREIGN KEY (workspace_id, plant_id, moment_id)
        REFERENCES moments (workspace_id, plant_id, id) ON DELETE CASCADE,
    -- The hub refuses to delete a file while a photo still names it.
    CONSTRAINT photos_file_fkey FOREIGN KEY (file_id)
        REFERENCES public.workspace_files (id) ON DELETE RESTRICT,
    -- GARDEN-INV-005: A file is the photo of at most one moment.
    CONSTRAINT photos_file_id_key UNIQUE (file_id)
);

CREATE INDEX photos_moment_idx ON photos (workspace_id, moment_id, id);

-- GARDEN-INV-006: The profile picture of a plant is a photo of that plant. The delete
-- of the photo clears the column alone, so the row keeps its workspace.
ALTER TABLE plants
    ADD COLUMN profile_photo_id UUID,
    ADD CONSTRAINT plants_profile_photo_fkey FOREIGN KEY (workspace_id, id, profile_photo_id)
        REFERENCES photos (workspace_id, plant_id, id) ON DELETE SET NULL (profile_photo_id);

CREATE TRIGGER set_updated_at
    BEFORE UPDATE ON moments
    FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER set_updated_at
    BEFORE UPDATE ON photos
    FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

ALTER TABLE moments
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE moments
    FORCE ROW LEVEL SECURITY;
CREATE POLICY moments_workspace_isolation ON moments
    USING (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid)
    WITH CHECK (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid);

ALTER TABLE photos
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE photos
    FORCE ROW LEVEL SECURITY;
CREATE POLICY photos_workspace_isolation ON photos
    USING (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid)
    WITH CHECK (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid);

COMMIT;
