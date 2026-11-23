BEGIN;

ALTER TABLE display_counters
    DROP CONSTRAINT display_counters_workspace_id_fkey,
    ADD CONSTRAINT display_counters_workspace_id_fkey FOREIGN KEY (workspace_id)
        REFERENCES public.workspaces (id);

ALTER TABLE environments
    DROP CONSTRAINT environments_workspace_id_fkey,
    ADD CONSTRAINT environments_workspace_id_fkey FOREIGN KEY (workspace_id)
        REFERENCES public.workspaces (id);

ALTER TABLE plants
    DROP CONSTRAINT plants_workspace_id_fkey,
    ADD CONSTRAINT plants_workspace_id_fkey FOREIGN KEY (workspace_id)
        REFERENCES public.workspaces (id);

ALTER TABLE care_intervals
    DROP CONSTRAINT care_intervals_workspace_id_fkey,
    ADD CONSTRAINT care_intervals_workspace_id_fkey FOREIGN KEY (workspace_id)
        REFERENCES public.workspaces (id);

ALTER TABLE maintenance_entries
    DROP CONSTRAINT maintenance_entries_workspace_id_fkey,
    ADD CONSTRAINT maintenance_entries_workspace_id_fkey FOREIGN KEY (workspace_id)
        REFERENCES public.workspaces (id);

ALTER TABLE moments
    DROP CONSTRAINT moments_workspace_id_fkey,
    ADD CONSTRAINT moments_workspace_id_fkey FOREIGN KEY (workspace_id)
        REFERENCES public.workspaces (id);

ALTER TABLE photos
    DROP CONSTRAINT photos_workspace_id_fkey,
    ADD CONSTRAINT photos_workspace_id_fkey FOREIGN KEY (workspace_id)
        REFERENCES public.workspaces (id);

COMMIT;
