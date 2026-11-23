BEGIN;

-- An older build has no handler of the type, and the outbox retries such a task forever.

DELETE FROM public.outbox_tasks WHERE type = 'secret_key.create';

COMMIT;
