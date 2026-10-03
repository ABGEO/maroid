BEGIN;

ALTER TABLE public.users
    ADD COLUMN is_administrator BOOLEAN NOT NULL DEFAULT false;

COMMIT;
