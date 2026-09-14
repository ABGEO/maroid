import { browser } from '$app/environment';
import { env } from '$env/dynamic/public';

import { createInstance, getInstance, type ModuleFederation } from '@module-federation/runtime';
import type { RouteMounter } from '@maroid/plugin-sdk';

export type { RouteCleanup, RouteMounter } from '@maroid/plugin-sdk';

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

export async function loadPluginMount(pluginId: string, subpath: string): Promise<RouteMounter> {
	if (!browser) {
		throw new Error('Plugin federation runtime is browser-only');
	}

	const name = ensureRegistered(pluginId);

	const mod = await federation().loadRemote<{ routes: unknown }>(`${name}/entry`);
	if (!mod || !isRouteManifest(mod.routes)) {
		throw new Error(`Plugin "${pluginId}" entry did not export a routes manifest`);
	}

	const key = '/' + subpath.replace(/^\/+/, '').replace(/\/+$/, '');
	const mounter = mod.routes[key];
	if (!mounter) {
		const available = Object.keys(mod.routes).join(', ') || '(none)';

		throw new Error(`Plugin "${pluginId}" has no route for "${key}". Available: ${available}`);
	}

	return mounter;
}
