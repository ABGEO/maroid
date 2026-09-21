import { resolve } from '$app/paths';
import type { ResolvedPathname, RouteId } from '$app/types';

import type { Plugin } from '$lib/api/types';
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
}

const HUB: Crumb = { label: 'hub', href: resolve('/') };

const PLUGINS: Crumb = { label: 'Plugins', href: resolve('/plugins') };

/**
 * Reports whether a trail needs the loaded plugins to read correctly. A caller
 * waits for them before it shows the trail of a plugin route.
 */
export function needsPlugins(routeId: RouteId | null): boolean {
	return (
		routeId === '/(dashboard)/plugins/[plugin]/settings' ||
		routeId === '/(dashboard)/plugins/[plugin]/[...path]'
	);
}

/**
 * The trail for one location. The hub opens every trail, and the last crumb is
 * the current page. An unknown route answers the hub alone.
 */
export function crumbsFor(
	routeId: RouteId | null,
	params: CrumbParams,
	plugins: Plugin[]
): Crumb[] {
	switch (routeId) {
		case '/(dashboard)/profile':
			return [HUB, { label: 'Profile', href: resolve('/profile') }];

		case '/(dashboard)/plugins':
			return [HUB, PLUGINS];

		case '/(dashboard)/plugins/[plugin]/settings': {
			const pluginId = params.plugin ?? '';

			return [
				HUB,
				PLUGINS,
				pluginCrumb(pluginId, plugins),
				{
					label: 'Settings',
					href: resolve('/(dashboard)/plugins/[plugin]/settings', { plugin: pluginId })
				}
			];
		}

		case '/(dashboard)/plugins/[plugin]/[...path]': {
			const pluginId = params.plugin ?? '';

			return [
				HUB,
				PLUGINS,
				pluginCrumb(pluginId, plugins),
				...routeCrumbs(pluginId, params.path ?? '', plugins)
			];
		}

		default:
			return [HUB];
	}
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
function routeCrumbs(pluginId: string, path: string, plugins: Plugin[]): Crumb[] {
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
			href: resolve('/(dashboard)/plugins/[plugin]/[...path]', { plugin: pluginId, path: subpath })
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
