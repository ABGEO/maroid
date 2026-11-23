CREATE TABLE probe.places
(
    id           UUID NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    workspace_id UUID NOT NULL
        DEFAULT NULLIF(current_setting('app.workspace_id', true), '')::uuid
        REFERENCES public.workspaces (id) ON DELETE CASCADE,
    name         TEXT NOT NULL
);

ALTER TABLE probe.places ENABLE ROW LEVEL SECURITY;
ALTER TABLE probe.places FORCE ROW LEVEL SECURITY;

CREATE POLICY places_workspace_isolation ON probe.places
    USING (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid)
    WITH CHECK (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid);

CREATE TABLE probe.photos
(
    id           UUID NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    workspace_id UUID NOT NULL
        DEFAULT NULLIF(current_setting('app.workspace_id', true), '')::uuid
        REFERENCES public.workspaces (id) ON DELETE CASCADE,
    place_id     UUID NOT NULL REFERENCES probe.places (id),
    file_id      UUID NOT NULL REFERENCES public.workspace_files (id) ON DELETE RESTRICT
);

ALTER TABLE probe.photos ENABLE ROW LEVEL SECURITY;
ALTER TABLE probe.photos FORCE ROW LEVEL SECURITY;

CREATE POLICY photos_workspace_isolation ON probe.photos
    USING (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid)
    WITH CHECK (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid);
