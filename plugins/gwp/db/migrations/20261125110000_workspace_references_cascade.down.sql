BEGIN;

ALTER TABLE customer_assignments
    DROP CONSTRAINT customer_assignments_workspace_id_fkey,
    ADD CONSTRAINT customer_assignments_workspace_id_fkey FOREIGN KEY (workspace_id)
        REFERENCES public.workspaces (id);

ALTER TABLE readings
    DROP CONSTRAINT readings_workspace_id_fkey,
    ADD CONSTRAINT readings_workspace_id_fkey FOREIGN KEY (workspace_id)
        REFERENCES public.workspaces (id);

COMMIT;
