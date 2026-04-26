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
