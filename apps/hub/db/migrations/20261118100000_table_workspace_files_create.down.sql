BEGIN;

DROP TABLE IF EXISTS public.workspace_files;
DROP FUNCTION IF EXISTS public.files_keep_identity();

COMMIT;
