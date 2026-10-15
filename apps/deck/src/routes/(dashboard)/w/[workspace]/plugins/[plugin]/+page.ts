import { error } from '@sveltejs/kit';

import { ApiError, api, type EnabledPlugin } from '$lib/api';
import { PERMISSION, holds } from '$lib/permissions';
import { loadPlugins, pluginState } from '$lib/state/plugins.svelte';

import type { PageLoad } from './$types';

const NOT_FOUND = 'This workspace has no plugin at this address.';

async function readEnablement(
	workspaceId: string,
	pluginId: string
): Promise<EnabledPlugin | null> {
	try {
		return await api.enablements.get(workspaceId, pluginId);
	} catch (failure) {
		if (failure instanceof ApiError && failure.status === 404) {
			return null;
		}

		throw failure;
	}
}

export const load: PageLoad = async ({ params, parent, depends }) => {
	depends('app:plugin-detail');

	const { workspace } = await parent();
	const pluginId = params.plugin;

	const enablement = await readEnablement(workspace.id, pluginId);
	if (enablement !== null) {
		return { pluginId, enablement };
	}

	// A plugin that the workspace does not use is open only to a person who may turn it on.
	if (!holds(workspace, PERMISSION.pluginsWrite)) {
		error(404, NOT_FOUND);
	}

	await loadPlugins();
	if (pluginState.status === 'error') {
		throw new Error('The hub answered no plugin');
	}

	if (!pluginState.plugins.some((plugin) => plugin.id === pluginId)) {
		error(404, NOT_FOUND);
	}

	return { pluginId, enablement: null };
};
