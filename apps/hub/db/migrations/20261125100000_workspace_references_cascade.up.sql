BEGIN;

-- WSREMOVE-INV-001: No record names a workspace that does not exist. Under OWN-005 the
-- delete of a workspace deletes every row that it scopes. The cascade runs as the owner
-- of each table, so the policy of a table hides no row from it.

ALTER TABLE public.workspace_members
    DROP CONSTRAINT workspace_members_workspace_id_fkey,
    ADD CONSTRAINT workspace_members_workspace_id_fkey FOREIGN KEY (workspace_id)
        REFERENCES public.workspaces (id) ON DELETE CASCADE;

ALTER TABLE public.workspace_plugins
    DROP CONSTRAINT workspace_plugins_workspace_id_fkey,
    ADD CONSTRAINT workspace_plugins_workspace_id_fkey FOREIGN KEY (workspace_id)
        REFERENCES public.workspaces (id) ON DELETE CASCADE;

ALTER TABLE public.plugin_workspace_settings
    DROP CONSTRAINT plugin_workspace_settings_workspace_id_fkey,
    ADD CONSTRAINT plugin_workspace_settings_workspace_id_fkey FOREIGN KEY (workspace_id)
        REFERENCES public.workspaces (id) ON DELETE CASCADE;

ALTER TABLE public.plugin_workspace_secrets
    DROP CONSTRAINT plugin_workspace_secrets_workspace_id_fkey,
    ADD CONSTRAINT plugin_workspace_secrets_workspace_id_fkey FOREIGN KEY (workspace_id)
        REFERENCES public.workspaces (id) ON DELETE CASCADE;

ALTER TABLE public.notification_providers
    DROP CONSTRAINT notification_providers_workspace_id_fkey,
    ADD CONSTRAINT notification_providers_workspace_id_fkey FOREIGN KEY (workspace_id)
        REFERENCES public.workspaces (id) ON DELETE CASCADE;

ALTER TABLE public.notification_routes
    DROP CONSTRAINT notification_routes_workspace_id_fkey,
    ADD CONSTRAINT notification_routes_workspace_id_fkey FOREIGN KEY (workspace_id)
        REFERENCES public.workspaces (id) ON DELETE CASCADE;

ALTER TABLE public.workspace_files
    DROP CONSTRAINT workspace_files_workspace_id_fkey,
    ADD CONSTRAINT workspace_files_workspace_id_fkey FOREIGN KEY (workspace_id)
        REFERENCES public.workspaces (id) ON DELETE CASCADE;

-- WSREMOVE-FR-015: A chat belongs to its user and stays. It only forgets the selection.
ALTER TABLE public.telegram_chats
    DROP CONSTRAINT telegram_chats_selected_workspace_id_fkey,
    ADD CONSTRAINT telegram_chats_selected_workspace_id_fkey FOREIGN KEY (selected_workspace_id)
        REFERENCES public.workspaces (id) ON DELETE SET NULL;

COMMIT;
