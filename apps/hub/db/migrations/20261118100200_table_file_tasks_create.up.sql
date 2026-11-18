BEGIN;

-- A shared table, as a queue of the hub: the worker reads it with no acting user and no
-- acting workspace, and runs each task in the scope of the owner that the row names.
-- A row holds identifiers and no content of a file.
CREATE TABLE public.file_tasks
(
    id           UUID        NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    -- A purge outlives the row of its file, so the column holds no foreign key.
    file_id      UUID        NOT NULL,
    scope        TEXT        NOT NULL,
    owner_id     UUID        NOT NULL,
    object_key   TEXT        NOT NULL,
    action       TEXT        NOT NULL,
    run_after    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    locked_until TIMESTAMPTZ,
    attempts     INTEGER     NOT NULL DEFAULT 0,
    last_error   TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT file_tasks_scope_check CHECK (scope IN ('workspace', 'user')),
    CONSTRAINT file_tasks_action_check
        CHECK (action IN ('process', 'check', 'discard_upload', 'expire', 'purge'))
);

CREATE INDEX file_tasks_run_after
    ON public.file_tasks (run_after);

COMMIT;
