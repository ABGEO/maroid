BEGIN;

CREATE TABLE public.idempotency_keys
(
    id           UUID        NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    user_id      UUID        NOT NULL
        DEFAULT NULLIF(current_setting('app.user_id', true), '')::uuid
        REFERENCES public.users (id) ON DELETE CASCADE,
    key          TEXT        NOT NULL,
    request_hash TEXT        NOT NULL,
    status       SMALLINT    NOT NULL,
    headers      JSONB       NOT NULL             DEFAULT '{}'::jsonb,
    body         BYTEA       NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL             DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL             DEFAULT now(),
    CONSTRAINT idempotency_keys_user_key UNIQUE (user_id, key)
);

ALTER TABLE public.idempotency_keys
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.idempotency_keys
    FORCE ROW LEVEL SECURITY;

CREATE POLICY idempotency_keys_user_isolation ON public.idempotency_keys
    USING (user_id = NULLIF(current_setting('app.user_id', true), '')::uuid)
    WITH CHECK (user_id = NULLIF(current_setting('app.user_id', true), '')::uuid);

CREATE INDEX idempotency_keys_created_at ON public.idempotency_keys (created_at);

CREATE TRIGGER set_updated_at
    BEFORE UPDATE
    ON public.idempotency_keys
    FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

COMMIT;
