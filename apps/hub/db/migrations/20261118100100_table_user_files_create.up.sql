BEGIN;

-- A user scopes the table, because a file of a user kind belongs to one person in every
-- workspace. The owner is the uploader, so the table holds no uploaded_by.
CREATE TABLE public.user_files
(
    id         UUID        NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    user_id    UUID        NOT NULL
        DEFAULT NULLIF(current_setting('app.user_id', true), '')::uuid
        REFERENCES public.users (id),
    plugin_id  TEXT        NOT NULL,
    kind       TEXT        NOT NULL,
    state      TEXT        NOT NULL             DEFAULT 'pending',
    failure    TEXT,
    name       TEXT        NOT NULL,
    media_type TEXT        NOT NULL,
    size       BIGINT      NOT NULL,
    width      INTEGER,
    height     INTEGER,
    claimed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL             DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL             DEFAULT NOW(),
    CONSTRAINT user_files_kind_check CHECK (kind ~ '^[a-z0-9-]{1,64}$'),
    CONSTRAINT user_files_state_check
        CHECK (state IN ('pending', 'processing', 'ready', 'failed')),
    CONSTRAINT user_files_failure_check
        CHECK (failure IN ('media_type', 'damaged', 'bytes_absent')),
    -- FILES-FR-011: A failed file names its reason, and no other file names one.
    CONSTRAINT user_files_failure_state_check CHECK ((state = 'failed') = (failure IS NOT NULL)),
    CONSTRAINT user_files_name_check CHECK (char_length(name) BETWEEN 1 AND 255),
    -- FILES-FR-005: No kind accepts more than 25 MB.
    CONSTRAINT user_files_size_check CHECK (size > 0 AND size <= 26214400)
);

CREATE INDEX user_files_user_plugin
    ON public.user_files (user_id, plugin_id);

ALTER TABLE public.user_files
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.user_files
    FORCE ROW LEVEL SECURITY;

-- FILES-INV-003: A file of one user reaches no other user.
CREATE POLICY user_files_user_isolation ON public.user_files
    USING (user_id = NULLIF(current_setting('app.user_id', true), '')::uuid)
    WITH CHECK (user_id = NULLIF(current_setting('app.user_id', true), '')::uuid);

CREATE TRIGGER files_keep_identity
    BEFORE UPDATE
    ON public.user_files
    FOR EACH ROW
EXECUTE FUNCTION public.files_keep_identity('user_id');

CREATE TRIGGER set_updated_at
    BEFORE UPDATE
    ON public.user_files
    FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

COMMIT;
