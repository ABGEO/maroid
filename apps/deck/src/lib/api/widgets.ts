import type { Page } from '@maroid/api-client';

import { client } from './client';
import type { Widget, WidgetAnswer } from './types';

function path(workspaceId: string): string {
	return `/workspaces/${encodeURIComponent(workspaceId)}`;
}

export const widgets = {
	list: (workspaceId: string): Promise<Widget[] | null> =>
		client.get<Page<Widget>>(`${path(workspaceId)}/widgets`).then((page) => page?.items ?? null),

	read: (workspaceId: string, pluginId: string, key: string): Promise<WidgetAnswer | null> =>
		client.get<WidgetAnswer>(
			`${path(workspaceId)}/plugins/${encodeURIComponent(pluginId)}/widgets/${encodeURIComponent(key)}`
		)
};
