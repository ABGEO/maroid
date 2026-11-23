BEGIN;

-- A shared table, as a queue of the hub: the worker reads it with no acting user and no
-- acting workspace, and runs each task in the scope of the owner that the row names.
-- A payload holds identifiers and no content.
CREATE TABLE public.outbox_tasks
(
    id           UUID        NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    type         TEXT        NOT NULL,
    payload      JSONB       NOT NULL DEFAULT '{}'::jsonb,
    owner_scope  TEXT,
    owner_id     UUID,
    dedupe_key   TEXT,
    run_after    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    locked_until TIMESTAMPTZ,
    attempts     INTEGER     NOT NULL DEFAULT 0,
    last_error   TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- JOB-010: A type names a domain of the hub and an action.
    CONSTRAINT outbox_tasks_type_check CHECK (type ~ '^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)+$'),
    CONSTRAINT outbox_tasks_owner_scope_check CHECK (owner_scope IN ('workspace', 'user')),
    CONSTRAINT outbox_tasks_owner_check CHECK ((owner_scope IS NULL) = (owner_id IS NULL))
);

CREATE INDEX outbox_tasks_run_after ON public.outbox_tasks (run_after);

CREATE UNIQUE INDEX outbox_tasks_dedupe_key ON public.outbox_tasks (dedupe_key)
    WHERE dedupe_key IS NOT NULL;

INSERT INTO public.outbox_tasks (id, type, payload, owner_scope, owner_id, dedupe_key,
                                 run_after, locked_until, attempts, last_error, created_at)
SELECT id,
       'file.' || action,
       jsonb_build_object('file_id', file_id, 'object_key', object_key),
       scope,
       owner_id,
       CASE WHEN action = 'process' THEN 'file.process:' || file_id END,
       run_after,
       locked_until,
       attempts,
       last_error,
       created_at
FROM public.file_tasks
ON CONFLICT DO NOTHING;

DROP TABLE public.file_tasks;

COMMIT;
