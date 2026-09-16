BEGIN;

ALTER TABLE public.users
    ADD COLUMN telegram_id  BIGINT,
    ADD COLUMN username     TEXT,
    ADD COLUMN display_name TEXT,
    ADD COLUMN picture_url  TEXT;

UPDATE public.users u
SET telegram_id  = i.provider_user_id::BIGINT,
    username     = i.username,
    display_name = i.display_name,
    picture_url  = i.picture_url
FROM public.identities i
WHERE i.user_id = u.id
  AND i.provider = 'telegram';

-- The column holds the identity again, so the row that carried it goes. Without
-- this delete the migration cannot run a second time: the extract would insert a
-- pair that the unique constraint already holds.
DELETE FROM public.identities WHERE provider = 'telegram';

-- This shape cannot hold a record with no Telegram identity,
-- because telegram_id is NOT NULL. The loss is the cost of this direction.
DELETE FROM public.users WHERE telegram_id IS NULL;

ALTER TABLE public.users
    ALTER COLUMN telegram_id SET NOT NULL,
    DROP COLUMN first_name,
    DROP COLUMN last_name;

ALTER TABLE public.users
    ADD CONSTRAINT users_telegram_id_key UNIQUE (telegram_id);

COMMIT;
