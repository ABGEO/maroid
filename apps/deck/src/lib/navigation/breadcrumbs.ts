import { resolve } from '$app/paths';
import type { ResolvedPathname, RouteId } from '$app/types';

import type { Plugin, Workspace } from '$lib/api/types';
import { displayNameOf, uiOf } from '$lib/plugins/capabilities';

/**
 * One step of the trail. A crumb without a target names a place that carries no
 * page of its own, such as a plugin. A pending crumb waits for the page to name
 * itself.
 */
export interface Crumb {
	label: string;
	href?: ResolvedPathname;
	pending?: boolean;
}

/** What the mounted plugin page tells the trail about itself. */
export interface PageState {
	/** The name that the page gave itself, or null. */
	title: string | null;
	/** Whether the matched route of the page holds a parameter segment. */
	parameterized: boolean;
}

const NO_PAGE: PageState = { title: null, parameterized: false };

/** The parameters that a dashboard route can carry. */
export interface CrumbParams {
	plugin?: string;
	path?: string;
	provider?: string;
	user?: string;
	workspace?: string;
}

const HUB: Crumb = { label: 'hub', href: resolve('/') };

const ADMINISTRATION: Crumb = { label: 'Administration' };

const USERS: Crumb = { label: 'Users', href: resolve('/admin/users') };

const PROVIDERS: Crumb = { label: 'Sign-in providers', href: resolve('/admin/providers') };

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
	workspaces: Workspace[] = [],
	pageState: PageState = NO_PAGE
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

		case '/(dashboard)/admin/plugins':
			return [HUB, ADMINISTRATION, { label: 'Plugins', href: resolve('/admin/plugins') }];

		case '/(dashboard)/admin/providers':
			return [HUB, ADMINISTRATION, PROVIDERS];

		case '/(dashboard)/admin/providers/[provider]':
			return [
				HUB,
				ADMINISTRATION,
				PROVIDERS,
				{
					label: 'Provider',
					href: resolve('/(dashboard)/admin/providers/[provider]', {
						provider: params.provider ?? ''
					})
				}
			];

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

		case '/(dashboard)/w/[workspace]/settings/notifications': {
			const workspaceId = params.workspace ?? '';

			return [
				HUB,
				workspaceCrumb(workspaceId, workspaces),
				{
					label: 'Settings',
					href: resolve('/(dashboard)/w/[workspace]/settings', { workspace: workspaceId })
				},
				{
					label: 'Notifications',
					href: resolve('/(dashboard)/w/[workspace]/settings/notifications', {
						workspace: workspaceId
					})
				}
			];
		}

		case '/(dashboard)/profile':
			return [HUB, { label: 'Profile', href: resolve('/profile') }];

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
				...routeCrumbs(workspaceId, pluginId, params.path ?? '', plugins, pageState)
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
 * The crumbs of a plugin path: one link for each navigation entry that leads the
 * path, then the page itself. The page reads as the title it gave, as a pending
 * crumb while a page with a parameter has none, or as the entry or the last segment
 * of the path. A segment that no entry declares takes no crumb of its own.
 */
function routeCrumbs(
	workspaceId: string,
	pluginId: string,
	path: string,
	plugins: Plugin[],
	pageState: PageState
): Crumb[] {
	const routes = uiOf(find(plugins, pluginId))?.routes ?? [];
	const segments = path.split('/').filter(Boolean);
	if (segments.length === 0) {
		return [];
	}

	const crumbOf = (route: { path: string; label: string }): Crumb => ({
		label: route.label,
		href: resolve('/(dashboard)/w/[workspace]/plugins/[plugin]/[...path]', {
			workspace: workspaceId,
			plugin: pluginId,
			path: trim(route.path)
		})
	});

	const leading = routes
		.map((route) => ({ route, depth: trim(route.path).split('/').filter(Boolean).length }))
		.filter(
			({ route, depth }) =>
				depth < segments.length && trim(route.path) === segments.slice(0, depth).join('/')
		)
		.sort((left, right) => left.depth - right.depth)
		.map(({ route }) => crumbOf(route));

	const exact = routes.find((route) => trim(route.path) === segments.join('/'));

	let last: Crumb;
	if (pageState.title !== null) {
		last = { label: pageState.title };
	} else if (pageState.parameterized) {
		last = { label: '', pending: true };
	} else if (exact) {
		last = crumbOf(exact);
	} else {
		last = { label: titleOf(segments[segments.length - 1]) };
	}

	return [...leading, last];
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
