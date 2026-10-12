import type { Page } from '@maroid/api-client';

import { client } from './client';
import type { Widget, WidgetAnswer } from './types';

function path(workspaceId: string): string {
	return `/workspaces/${encodeURIComponent(workspaceId)}`;
}

export const widgets = {
	list: (workspaceId: string): Promise<Widget[] | null> =>
		client.get<Page<Widget>>(`${path(workspaceId)}/widgets`).then((page) => page?.items ?? null),

	read: async (workspaceId: string, pluginId: string, key: string): Promise<WidgetAnswer> => {
		const answer = await client.get<WidgetAnswer>(
			`${path(workspaceId)}/plugins/${encodeURIComponent(pluginId)}/widgets/${encodeURIComponent(key)}`
		);
		if (!answer) {
			throw new Error('the hub answered no widget');
		}

		return answer;
	}
};
