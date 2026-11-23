BEGIN;

ALTER TABLE customer_assignments
    DROP CONSTRAINT customer_assignments_workspace_id_fkey,
    ADD CONSTRAINT customer_assignments_workspace_id_fkey FOREIGN KEY (workspace_id)
        REFERENCES public.workspaces (id);

ALTER TABLE transactions
    DROP CONSTRAINT transactions_workspace_id_fkey,
    ADD CONSTRAINT transactions_workspace_id_fkey FOREIGN KEY (workspace_id)
        REFERENCES public.workspaces (id);

COMMIT;
