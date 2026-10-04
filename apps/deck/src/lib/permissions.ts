import type { Role, Workspace } from '$lib/api';

/** The permissions of the hub that a page of the deck checks. */
export const PERMISSION = {
	workspaceWrite: 'workspace.write',
	membersWrite: 'members.write',
	pluginsWrite: 'plugins.write',
	settingsWrite: 'settings.write'
} as const;

/** The roles, from the most access to the least. */
export const ROLES: readonly Role[] = ['manager', 'editor', 'viewer'];

/** Whether the role of the person in the workspace holds the permission. */
export function holds(workspace: Workspace, permission: string): boolean {
	return workspace.permissions?.includes(permission) ?? false;
}

/**
 * Whether the role of the person holds a permission of a plugin. The plugin names the
 * permission without its prefix, and the hub answers it with the prefix.
 */
export function pluginCan(workspace: Workspace, pluginId: string, permission: string): boolean {
	return holds(workspace, `${pluginId}:${permission}`);
}

/** The label of a role, for a person to read. */
export function roleLabel(role: Role): string {
	return role.charAt(0).toUpperCase() + role.slice(1);
}
