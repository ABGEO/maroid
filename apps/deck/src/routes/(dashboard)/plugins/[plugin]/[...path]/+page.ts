import type { PageLoad } from './$types';

import { loadPluginMount } from '$lib/plugins/loader';

export const load: PageLoad = async ({ params }) => {
	const { plugin, path = '' } = params;
	const mount = await loadPluginMount(plugin, path);

	return { mount, pluginId: plugin };
};
