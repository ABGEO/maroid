import { client } from './client';
import type { SettingsInput, SettingsSchema, SettingsValues } from './types';

function base(pluginId: string): string {
	return `/plugins/${encodeURIComponent(pluginId)}/settings`;
}

export const settings = {
	schema: (pluginId: string): Promise<SettingsSchema | null> =>
		client.get<SettingsSchema>(`${base(pluginId)}/schema`),

	read: (pluginId: string): Promise<SettingsValues | null> =>
		client.get<SettingsValues>(base(pluginId)),

	save: (pluginId: string, input: SettingsInput): Promise<void> =>
		client.put<void>(base(pluginId), input).then(() => undefined)
};
