BEGIN;

ALTER TABLE billing_items
    DROP CONSTRAINT billing_items_workspace_id_fkey,
    ADD CONSTRAINT billing_items_workspace_id_fkey FOREIGN KEY (workspace_id)
        REFERENCES public.workspaces (id);

COMMIT;
