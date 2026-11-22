BEGIN;

-- A row lives 30 days, so a claim ends no later than its message.
CREATE TABLE public.telegram_messages
(
    id          UUID        NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    user_id     UUID        NOT NULL
        DEFAULT NULLIF(current_setting('app.user_id', true), '')::uuid
        REFERENCES public.users (id) ON DELETE CASCADE,
    plugin_id   TEXT        NOT NULL,
    chat_id     BIGINT      NOT NULL,
    message_id  BIGINT      NOT NULL,
    claim_until TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT telegram_messages_chat_message_key UNIQUE (chat_id, message_id),
    CONSTRAINT telegram_messages_claim_check
        CHECK (claim_until <= created_at + INTERVAL '30 days')
);

CREATE INDEX telegram_messages_open_claim_idx
    ON public.telegram_messages (user_id, created_at)
    WHERE claim_until IS NOT NULL;

ALTER TABLE public.telegram_messages
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.telegram_messages
    FORCE ROW LEVEL SECURITY;

CREATE POLICY telegram_messages_user_isolation ON public.telegram_messages
    USING (user_id = NULLIF(current_setting('app.user_id', true), '')::uuid)
    WITH CHECK (user_id = NULLIF(current_setting('app.user_id', true), '')::uuid);

CREATE TRIGGER set_updated_at
    BEFORE UPDATE
    ON public.telegram_messages
    FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

COMMIT;
