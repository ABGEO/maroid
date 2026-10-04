import { api, type Plugin } from '$lib/api';

export type PluginStatus = 'idle' | 'loading' | 'ready' | 'error';

export const pluginState = $state<{ status: PluginStatus; plugins: Plugin[] }>({
	status: 'idle',
	plugins: []
});

export async function loadPlugins(): Promise<void> {
	if (pluginState.status !== 'idle') {
		return;
	}

	pluginState.status = 'loading';

	try {
		const list = await api.plugins.list();
		if (list === null) {
			return;
		}

		pluginState.plugins = list;
		pluginState.status = 'ready';
	} catch (error) {
		console.error('Failed to load plugins', error);
		pluginState.status = 'error';
		pluginState.plugins = [];
	}
}

/**
 * The entries of the plugins that one workspace enables. The navigation names these
 * alone, whatever the allowlist of the person holds.
 */
export const enabledState = $state<{
	workspaceId: string | null;
	status: PluginStatus;
	plugins: Plugin[];
}>({
	workspaceId: null,
	status: 'idle',
	plugins: []
});

/**
 * Loads the plugins that the workspace enables. A call for the workspace it already
 * holds reloads it only when `force` is set, as after a switch on the page of the plugins.
 */
export async function loadEnabled(workspaceId: string, force = false): Promise<void> {
	if (!force && enabledState.workspaceId === workspaceId && enabledState.status !== 'error') {
		return;
	}

	enabledState.workspaceId = workspaceId;
	enabledState.status = 'loading';

	try {
		const list = await api.enablements.list(workspaceId);
		if (list === null || enabledState.workspaceId !== workspaceId) {
			return;
		}

		enabledState.plugins = list.flatMap((enabled) => (enabled.plugin ? [enabled.plugin] : []));
		enabledState.status = 'ready';
	} catch (error) {
		console.error('Failed to load the plugins of the workspace', error);

		if (enabledState.workspaceId === workspaceId) {
			enabledState.plugins = [];
			enabledState.status = 'error';
		}
	}
}

/** Whether the workspace that the state holds enables the plugin. */
export function isEnabled(workspaceId: string, pluginId: string): boolean {
	return (
		enabledState.workspaceId === workspaceId &&
		enabledState.plugins.some((plugin) => plugin.id === pluginId)
	);
}

/**
 * The entry of a plugin: from the plugins of the workspace first, then from the
 * plugins that the person can turn on.
 */
export function knownPlugin(pluginId: string): Plugin | undefined {
	return (
		enabledState.plugins.find((plugin) => plugin.id === pluginId) ??
		pluginState.plugins.find((plugin) => plugin.id === pluginId)
	);
}
