import type { Page } from '@maroid/api-client';

import { client } from './client';
import type { EnabledPlugin } from './types';

function path(workspaceId: string, pluginId?: string): string {
	const base = `/workspaces/${encodeURIComponent(workspaceId)}/plugins`;

	return pluginId === undefined ? base : `${base}/${encodeURIComponent(pluginId)}`;
}

export const enablements = {
	list: (workspaceId: string): Promise<EnabledPlugin[] | null> =>
		client.get<Page<EnabledPlugin>>(path(workspaceId)).then((page) => page?.items ?? null),

	enable: (workspaceId: string, pluginId: string): Promise<EnabledPlugin | null> =>
		client.put<EnabledPlugin>(path(workspaceId, pluginId)),

	disable: (workspaceId: string, pluginId: string): Promise<void> =>
		client.del<void>(path(workspaceId, pluginId)).then(() => undefined)
};
