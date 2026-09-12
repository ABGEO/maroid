BEGIN;


CREATE TABLE public.users
(
    id           UUID        NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    telegram_id  BIGINT      NOT NULL UNIQUE,
    username     TEXT,
    display_name TEXT,
    picture_url  TEXT,
    status       TEXT        NOT NULL DEFAULT 'active'
        CONSTRAINT users_status_check CHECK (status IN ('active', 'blocked')),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TRIGGER set_updated_at
    BEFORE UPDATE
    ON public.users
    FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

COMMIT;
