import type { Tagged } from '@maroid/api-client';

import { client } from './client';
import type { SettingsInput, SettingsSchema, SettingsValues } from './types';

function base(workspaceId: string, pluginId: string): string {
	return `/workspaces/${encodeURIComponent(workspaceId)}/plugins/${encodeURIComponent(pluginId)}/settings`;
}

export const settings = {
	schema: (workspaceId: string, pluginId: string): Promise<SettingsSchema | null> =>
		client.get<SettingsSchema>(`${base(workspaceId, pluginId)}/schema`),

	read: (workspaceId: string, pluginId: string): Promise<Tagged<SettingsValues> | null> =>
		client.getTagged<SettingsValues>(base(workspaceId, pluginId)),

	save: (
		workspaceId: string,
		pluginId: string,
		input: SettingsInput,
		ifMatch?: string
	): Promise<void> =>
		client.put<void>(base(workspaceId, pluginId), input, { ifMatch }).then(() => undefined)
};
