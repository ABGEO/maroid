BEGIN;

-- WSREMOVE-INV-001: No record names a workspace that does not exist. Under OWN-005 the
-- delete of a workspace deletes every row that it scopes. The cascade runs as the owner
-- of each table, so the policy of a table hides no row from it.

ALTER TABLE display_counters
    DROP CONSTRAINT display_counters_workspace_id_fkey,
    ADD CONSTRAINT display_counters_workspace_id_fkey FOREIGN KEY (workspace_id)
        REFERENCES public.workspaces (id) ON DELETE CASCADE;

ALTER TABLE environments
    DROP CONSTRAINT environments_workspace_id_fkey,
    ADD CONSTRAINT environments_workspace_id_fkey FOREIGN KEY (workspace_id)
        REFERENCES public.workspaces (id) ON DELETE CASCADE;

ALTER TABLE plants
    DROP CONSTRAINT plants_workspace_id_fkey,
    ADD CONSTRAINT plants_workspace_id_fkey FOREIGN KEY (workspace_id)
        REFERENCES public.workspaces (id) ON DELETE CASCADE;

ALTER TABLE care_intervals
    DROP CONSTRAINT care_intervals_workspace_id_fkey,
    ADD CONSTRAINT care_intervals_workspace_id_fkey FOREIGN KEY (workspace_id)
        REFERENCES public.workspaces (id) ON DELETE CASCADE;

ALTER TABLE maintenance_entries
    DROP CONSTRAINT maintenance_entries_workspace_id_fkey,
    ADD CONSTRAINT maintenance_entries_workspace_id_fkey FOREIGN KEY (workspace_id)
        REFERENCES public.workspaces (id) ON DELETE CASCADE;

ALTER TABLE moments
    DROP CONSTRAINT moments_workspace_id_fkey,
    ADD CONSTRAINT moments_workspace_id_fkey FOREIGN KEY (workspace_id)
        REFERENCES public.workspaces (id) ON DELETE CASCADE;

ALTER TABLE photos
    DROP CONSTRAINT photos_workspace_id_fkey,
    ADD CONSTRAINT photos_workspace_id_fkey FOREIGN KEY (workspace_id)
        REFERENCES public.workspaces (id) ON DELETE CASCADE;

COMMIT;
