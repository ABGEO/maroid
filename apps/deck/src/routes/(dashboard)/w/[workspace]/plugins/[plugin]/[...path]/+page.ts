import { resolve } from '$app/paths';

import { loadPluginRoute } from '$lib/plugins/loader';

import type { PageLoad } from './$types';

export const load: PageLoad = async ({ params, url }) => {
	const { workspace, plugin } = params;
	// The params of the route arrive decoded, so a value that held an encoded slash
	// would split. The matcher reads the raw address and decodes each segment.
	const prefix = resolve('/(dashboard)/w/[workspace]/plugins/[plugin]/[...path]', {
		workspace,
		plugin,
		path: ''
	});
	const route = await loadPluginRoute(plugin, url.pathname.slice(prefix.length));

	return { route, pluginId: plugin };
};
