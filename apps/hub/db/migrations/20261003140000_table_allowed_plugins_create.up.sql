BEGIN;

CREATE TABLE public.allowed_plugins
(
    user_id    UUID        NOT NULL REFERENCES public.users (id),
    plugin_id  TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, plugin_id)
);

CREATE TRIGGER set_updated_at
    BEFORE UPDATE
    ON public.allowed_plugins
    FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

COMMIT;
