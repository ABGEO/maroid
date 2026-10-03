BEGIN;

CREATE TABLE public.telegram_chats
(
    id                    UUID        NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    user_id               UUID        NOT NULL
        DEFAULT NULLIF(current_setting('app.user_id', true), '')::uuid
        REFERENCES public.users (id),
    chat_id               BIGINT      NOT NULL,
    selected_workspace_id UUID        REFERENCES public.workspaces (id),
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT telegram_chats_user_chat_key UNIQUE (user_id, chat_id)
);

ALTER TABLE public.telegram_chats
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.telegram_chats
    FORCE ROW LEVEL SECURITY;

CREATE POLICY telegram_chats_user_isolation ON public.telegram_chats
    USING (user_id = NULLIF(current_setting('app.user_id', true), '')::uuid)
    WITH CHECK (user_id = NULLIF(current_setting('app.user_id', true), '')::uuid);

CREATE TRIGGER set_updated_at
    BEFORE UPDATE
    ON public.telegram_chats
    FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

COMMIT;
