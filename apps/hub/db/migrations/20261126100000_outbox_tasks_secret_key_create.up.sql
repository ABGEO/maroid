BEGIN;

-- A user record or a workspace that exists before the hub creates its keys receives
-- the task of a new one. The dedupe key skips a record whose task already waits.

INSERT INTO public.outbox_tasks (type, payload, dedupe_key)
SELECT 'secret_key.create',
       jsonb_build_object('kind', 'user', 'id', id),
       'secret_key.create:maroid-user-' || id
FROM public.users
UNION ALL
SELECT 'secret_key.create',
       jsonb_build_object('kind', 'workspace', 'id', id),
       'secret_key.create:maroid-workspace-' || id
FROM public.workspaces
ON CONFLICT (dedupe_key) WHERE dedupe_key IS NOT NULL DO NOTHING;

COMMIT;
