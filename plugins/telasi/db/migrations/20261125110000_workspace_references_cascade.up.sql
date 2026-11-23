BEGIN;

-- WSREMOVE-INV-001: No record names a workspace that does not exist. Under OWN-005 the
-- delete of a workspace deletes every row that it scopes. The cascade runs as the owner
-- of each table, so the policy of a table hides no row from it.

ALTER TABLE billing_items
    DROP CONSTRAINT billing_items_workspace_id_fkey,
    ADD CONSTRAINT billing_items_workspace_id_fkey FOREIGN KEY (workspace_id)
        REFERENCES public.workspaces (id) ON DELETE CASCADE;

COMMIT;
