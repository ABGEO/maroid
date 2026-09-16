BEGIN;

CREATE TABLE public.identities
(
    id               UUID        NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    user_id          UUID        NOT NULL REFERENCES public.users (id) ON DELETE CASCADE,
    provider         TEXT        NOT NULL,
    provider_user_id TEXT        NOT NULL,
    username         TEXT,
    display_name     TEXT,
    picture_url      TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT identities_provider_user_key UNIQUE (provider, provider_user_id)
);

CREATE INDEX identities_user_id_idx ON public.identities (user_id);

CREATE TRIGGER set_updated_at
    BEFORE UPDATE
    ON public.identities
    FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

COMMIT;
