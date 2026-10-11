import { browser } from '$app/environment';
import { env } from '$env/dynamic/public';

import { createInstance, getInstance, type ModuleFederation } from '@module-federation/runtime';
import type { RouteMounter, RouteParams } from '@maroid/plugin-sdk';

import { matchRoute } from './routes';

export type { RouteCleanup, RouteMounter } from '@maroid/plugin-sdk';

/** The page that an address of a plugin opens, with the values of its parameters. */
export interface PluginRoute {
	mount: RouteMounter;
	params: RouteParams;
	parameterized: boolean;
}

const HUB = (env.PUBLIC_HUB_BASE_URL ?? '').replace(/\/+$/, '');

type RouteManifest = Record<string, RouteMounter>;

const registered = new Set<string>();

let host: ModuleFederation | null = null;

function federation(): ModuleFederation {
	host ??= getInstance() ?? createInstance({ name: 'deck_host', remotes: [] });

	return host;
}

function sanitize(pluginId: string): string {
	return pluginId.replace(/[^a-zA-Z0-9_]/g, '_');
}

function manifestUrl(pluginId: string): string {
	return `${HUB}/plugins/${encodeURIComponent(pluginId)}/ui/mf-manifest.json`;
}

function ensureRegistered(pluginId: string): string {
	const name = sanitize(pluginId);
	if (registered.has(name)) {
		return name;
	}

	federation().registerRemotes([
		{
			name,
			alias: name,
			entry: manifestUrl(pluginId)
		}
	]);
	registered.add(name);

	return name;
}

function isRouteManifest(value: unknown): value is RouteManifest {
	if (!value || typeof value !== 'object') {
		return false;
	}

	return Object.values(value).every((v) => typeof v === 'function');
}

/**
 * Loads the remote of the plugin and answers the page that the raw subpath opens,
 * or null when no page route of the remote matches it.
 */
export async function loadPluginRoute(
	pluginId: string,
	subpath: string
): Promise<PluginRoute | null> {
	if (!browser) {
		throw new Error('Plugin federation runtime is browser-only');
	}

	const name = ensureRegistered(pluginId);

	const mod = await federation().loadRemote<{ routes: unknown }>(`${name}/entry`);
	if (!mod || !isRouteManifest(mod.routes)) {
		throw new Error(`Plugin "${pluginId}" entry did not export a routes manifest`);
	}

	const matched = matchRoute(Object.keys(mod.routes), subpath);
	if (matched === null) {
		return null;
	}

	return {
		mount: mod.routes[matched.key],
		params: matched.params,
		parameterized: matched.parameterized
	};
}
