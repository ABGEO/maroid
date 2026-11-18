BEGIN;

-- FILES-INV-005: A file keeps its plugin, its kind, and its owner for life. The first
-- argument of the trigger names the column of the owner.
CREATE FUNCTION public.files_keep_identity()
    RETURNS TRIGGER
    LANGUAGE plpgsql
AS
$$
BEGIN
    IF NEW.plugin_id IS DISTINCT FROM OLD.plugin_id
        OR NEW.kind IS DISTINCT FROM OLD.kind
        OR to_jsonb(NEW) ->> TG_ARGV[0] IS DISTINCT FROM to_jsonb(OLD) ->> TG_ARGV[0] THEN
        RAISE EXCEPTION 'a file keeps its plugin, its kind, and its owner'
            USING ERRCODE = 'check_violation';
    END IF;

    RETURN NEW;
END;
$$;

-- A workspace scopes the table, because a file of a workspace kind serves every member.
CREATE TABLE public.workspace_files
(
    id           UUID        NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    workspace_id UUID        NOT NULL
        DEFAULT NULLIF(current_setting('app.workspace_id', true), '')::uuid
        REFERENCES public.workspaces (id),
    plugin_id    TEXT        NOT NULL,
    kind         TEXT        NOT NULL,
    state        TEXT        NOT NULL             DEFAULT 'pending',
    failure      TEXT,
    name         TEXT        NOT NULL,
    media_type   TEXT        NOT NULL,
    size         BIGINT      NOT NULL,
    width        INTEGER,
    height       INTEGER,
    uploaded_by  UUID        NOT NULL
        DEFAULT NULLIF(current_setting('app.user_id', true), '')::uuid
        REFERENCES public.users (id),
    claimed_at   TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL             DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL             DEFAULT NOW(),
    CONSTRAINT workspace_files_kind_check CHECK (kind ~ '^[a-z0-9-]{1,64}$'),
    CONSTRAINT workspace_files_state_check
        CHECK (state IN ('pending', 'processing', 'ready', 'failed')),
    CONSTRAINT workspace_files_failure_check
        CHECK (failure IN ('media_type', 'damaged', 'bytes_absent')),
    -- FILES-FR-011: A failed file names its reason, and no other file names one.
    CONSTRAINT workspace_files_failure_state_check CHECK ((state = 'failed') = (failure IS NOT NULL)),
    CONSTRAINT workspace_files_name_check CHECK (char_length(name) BETWEEN 1 AND 255),
    -- FILES-FR-005: No kind accepts more than 25 MB.
    CONSTRAINT workspace_files_size_check CHECK (size > 0 AND size <= 26214400)
);

CREATE INDEX workspace_files_workspace_plugin
    ON public.workspace_files (workspace_id, plugin_id);

ALTER TABLE public.workspace_files
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.workspace_files
    FORCE ROW LEVEL SECURITY;

-- FILES-INV-003: A file of one workspace reaches no other workspace.
CREATE POLICY workspace_files_workspace_isolation ON public.workspace_files
    USING (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid)
    WITH CHECK (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid);

CREATE TRIGGER files_keep_identity
    BEFORE UPDATE
    ON public.workspace_files
    FOR EACH ROW
EXECUTE FUNCTION public.files_keep_identity('workspace_id');

CREATE TRIGGER set_updated_at
    BEFORE UPDATE
    ON public.workspace_files
    FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

COMMIT;
