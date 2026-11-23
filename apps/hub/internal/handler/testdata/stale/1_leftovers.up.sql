CREATE TABLE stale.leftovers
(
    id           UUID NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    workspace_id UUID NOT NULL REFERENCES public.workspaces (id)
);
