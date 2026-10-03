BEGIN;

ALTER TABLE public.workspace_members
    ADD COLUMN role TEXT NOT NULL DEFAULT 'manager'
        CONSTRAINT workspace_members_role_check CHECK (role IN ('manager', 'editor', 'viewer'));

ALTER TABLE public.workspace_members
    ALTER COLUMN role DROP DEFAULT;

COMMIT;
