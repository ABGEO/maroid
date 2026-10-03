BEGIN;

ALTER TABLE public.users
    DROP COLUMN IF EXISTS is_administrator;

COMMIT;
