import { browser } from '$app/environment';
import { PUBLIC_HUB_BASE_URL } from '$env/static/public';

import { loadRemote, registerRemotes } from '@module-federation/runtime';

const HUB = PUBLIC_HUB_BASE_URL.replace(/\/+$/, '');

export type RouteCleanup = () => void;
export type RouteMounter = (target: HTMLElement) => RouteCleanup;

type RouteManifest = Record<string, RouteMounter>;

const registered = new Set<string>();

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

	registerRemotes([
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

	const mod = await loadRemote<{ routes: unknown }>(`${name}/entry`);
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
