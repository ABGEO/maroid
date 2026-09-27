import type { Tagged } from '@maroid/api-client';

import { client } from './client';
import type { SettingsInput, SettingsSchema, SettingsValues } from './types';

function base(pluginId: string): string {
	return `/plugins/${encodeURIComponent(pluginId)}/settings`;
}

export const settings = {
	schema: (pluginId: string): Promise<SettingsSchema | null> =>
		client.get<SettingsSchema>(`${base(pluginId)}/schema`),

	read: (pluginId: string): Promise<Tagged<SettingsValues> | null> =>
		client.getTagged<SettingsValues>(base(pluginId)),

	save: (pluginId: string, input: SettingsInput, ifMatch?: string): Promise<void> =>
		client.put<void>(base(pluginId), input, { ifMatch }).then(() => undefined)
};
