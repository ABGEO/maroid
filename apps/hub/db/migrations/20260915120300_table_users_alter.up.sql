BEGIN;

-- users.telegram_id is the only place that names the external
-- account of a person who already signs in. The extract runs before the drop,
-- and in the same transaction, so a failure leaves both.
INSERT INTO public.identities (user_id, provider, provider_user_id, username, display_name, picture_url)
SELECT id, 'telegram', telegram_id::TEXT, username, display_name, picture_url
FROM public.users;

ALTER TABLE public.users
    ADD COLUMN first_name TEXT,
    ADD COLUMN last_name  TEXT;

-- The first space is the delimiter. The first word becomes the first name, and
-- the rest becomes the last name. A value with no space gives no last name.
UPDATE public.users
SET first_name = NULLIF(regexp_replace(trim(display_name), '\s.*$', ''), ''),
    last_name  = NULLIF(regexp_replace(trim(display_name), '^\S+\s*', ''), '');

ALTER TABLE public.users
    DROP COLUMN telegram_id,
    DROP COLUMN username,
    DROP COLUMN display_name,
    DROP COLUMN picture_url;

COMMIT;
