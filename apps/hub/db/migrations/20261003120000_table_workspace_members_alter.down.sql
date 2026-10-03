BEGIN;

ALTER TABLE public.workspace_members
    DROP COLUMN IF EXISTS role;

COMMIT;
