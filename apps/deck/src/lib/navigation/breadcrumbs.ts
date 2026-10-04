import { resolve } from '$app/paths';
import type { ResolvedPathname, RouteId } from '$app/types';

import type { Plugin, Workspace } from '$lib/api/types';
import { displayNameOf, uiOf } from '$lib/plugins/capabilities';

/**
 * One step of the trail. A crumb without a target names a place that carries no
 * page of its own, such as a plugin.
 */
export interface Crumb {
	label: string;
	href?: ResolvedPathname;
}

/** The parameters that a dashboard route can carry. */
export interface CrumbParams {
	plugin?: string;
	path?: string;
	user?: string;
	workspace?: string;
}

const HUB: Crumb = { label: 'hub', href: resolve('/') };

const PLUGINS: Crumb = { label: 'Plugins', href: resolve('/plugins') };

const ADMINISTRATION: Crumb = { label: 'Administration' };

const USERS: Crumb = { label: 'Users', href: resolve('/admin/users') };

/**
 * Reports whether a trail needs the loaded plugins to read correctly. A caller
 * waits for them before it shows the trail of a plugin route.
 */
export function needsPlugins(routeId: RouteId | null): boolean {
	return (
		routeId === '/(dashboard)/w/[workspace]/plugins/[plugin]/settings' ||
		routeId === '/(dashboard)/w/[workspace]/plugins/[plugin]/[...path]'
	);
}

/**
 * The trail for one location. The hub opens every trail, and the last crumb is
 * the current page. An unknown route answers the hub alone.
 */
export function crumbsFor(
	routeId: RouteId | null,
	params: CrumbParams,
	plugins: Plugin[],
	workspaces: Workspace[] = []
): Crumb[] {
	switch (routeId) {
		case '/(dashboard)/workspaces/new':
			return [HUB, { label: 'New workspace', href: resolve('/workspaces/new') }];

		case '/(dashboard)/w/[workspace]':
			return [HUB, workspaceCrumb(params.workspace ?? '', workspaces)];

		case '/(dashboard)/w/[workspace]/members': {
			const workspaceId = params.workspace ?? '';

			return [
				HUB,
				workspaceCrumb(workspaceId, workspaces),
				{
					label: 'Members',
					href: resolve('/(dashboard)/w/[workspace]/members', { workspace: workspaceId })
				}
			];
		}

		case '/(dashboard)/w/[workspace]/plugins': {
			const workspaceId = params.workspace ?? '';

			return [
				HUB,
				workspaceCrumb(workspaceId, workspaces),
				{
					label: 'Plugins',
					href: resolve('/(dashboard)/w/[workspace]/plugins', { workspace: workspaceId })
				}
			];
		}

		case '/(dashboard)/admin/users':
			return [HUB, ADMINISTRATION, USERS];

		case '/(dashboard)/admin/users/[user]':
			return [
				HUB,
				ADMINISTRATION,
				USERS,
				{
					label: 'User',
					href: resolve('/(dashboard)/admin/users/[user]', { user: params.user ?? '' })
				}
			];

		case '/(dashboard)/admin/workspaces':
			return [HUB, ADMINISTRATION, { label: 'Workspaces', href: resolve('/admin/workspaces') }];

		case '/(dashboard)/w/[workspace]/settings': {
			const workspaceId = params.workspace ?? '';

			return [
				HUB,
				workspaceCrumb(workspaceId, workspaces),
				{
					label: 'Settings',
					href: resolve('/(dashboard)/w/[workspace]/settings', { workspace: workspaceId })
				}
			];
		}

		case '/(dashboard)/profile':
			return [HUB, { label: 'Profile', href: resolve('/profile') }];

		case '/(dashboard)/plugins':
			return [HUB, PLUGINS];

		case '/(dashboard)/w/[workspace]/plugins/[plugin]/settings': {
			const workspaceId = params.workspace ?? '';
			const pluginId = params.plugin ?? '';

			return [
				HUB,
				workspaceCrumb(workspaceId, workspaces),
				pluginCrumb(pluginId, plugins),
				{
					label: 'Settings',
					href: resolve('/(dashboard)/w/[workspace]/plugins/[plugin]/settings', {
						workspace: workspaceId,
						plugin: pluginId
					})
				}
			];
		}

		case '/(dashboard)/w/[workspace]/plugins/[plugin]/[...path]': {
			const workspaceId = params.workspace ?? '';
			const pluginId = params.plugin ?? '';

			return [
				HUB,
				workspaceCrumb(workspaceId, workspaces),
				pluginCrumb(pluginId, plugins),
				...routeCrumbs(workspaceId, pluginId, params.path ?? '', plugins)
			];
		}

		default:
			return [HUB];
	}
}

/** The workspace, by its name once the list of workspaces arrived. */
function workspaceCrumb(workspaceId: string, workspaces: Workspace[]): Crumb {
	const workspace = workspaces.find((candidate) => candidate.id === workspaceId);

	return {
		label: workspace?.name ?? 'Workspace',
		href: resolve('/(dashboard)/w/[workspace]', { workspace: workspaceId })
	};
}

/** The plugin itself. The hub serves no page for it, so the crumb carries no target. */
function pluginCrumb(pluginId: string, plugins: Plugin[]): Crumb {
	return { label: displayNameOf(find(plugins, pluginId), pluginId) };
}

/**
 * One crumb for each segment of a plugin path. A segment that the manifest of
 * the plugin declares takes the label and the target of that route. A segment
 * that no route declares reads as a title and carries no target.
 */
function routeCrumbs(
	workspaceId: string,
	pluginId: string,
	path: string,
	plugins: Plugin[]
): Crumb[] {
	const routes = uiOf(find(plugins, pluginId))?.routes ?? [];
	const segments = path.split('/').filter(Boolean);

	return segments.map((segment, index) => {
		const subpath = segments.slice(0, index + 1).join('/');
		const route = routes.find((candidate) => trim(candidate.path) === subpath);

		if (!route) {
			return { label: titleOf(segment) };
		}

		return {
			label: route.label,
			href: resolve('/(dashboard)/w/[workspace]/plugins/[plugin]/[...path]', {
				workspace: workspaceId,
				plugin: pluginId,
				path: subpath
			})
		};
	});
}

function find(plugins: Plugin[], pluginId: string): Plugin | undefined {
	return plugins.find((plugin) => plugin.id === pluginId);
}

function trim(path: string): string {
	return path.replace(/^\/+/, '').replace(/\/+$/, '');
}

function titleOf(segment: string): string {
	return segment
		.split(/[-_]/)
		.filter(Boolean)
		.map((word) => word.charAt(0).toUpperCase() + word.slice(1))
		.join(' ');
}
