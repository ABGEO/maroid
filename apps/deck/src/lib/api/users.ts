import type { Page } from '@maroid/api-client';

import { client } from './client';
import type { Invitation, InvitedUser, PluginRef, UserRecord, UserStatus } from './types';

function path(userId: string): string {
	return `/users/${encodeURIComponent(userId)}`;
}

export interface NewUser {
	first_name?: string;
	last_name?: string;
	is_administrator: boolean;
	allowed_plugins: string[];
}

export interface UserChange {
	first_name?: string;
	last_name?: string;
	status?: UserStatus;
	is_administrator?: boolean;
}

export const users = {
	list: (): Promise<UserRecord[] | null> =>
		client.get<Page<UserRecord>>('/users').then((page) => page?.items ?? null),

	get: (userId: string): Promise<UserRecord | null> => client.get<UserRecord>(path(userId)),

	create: (user: NewUser, idempotencyKey: string): Promise<InvitedUser | null> =>
		client.post<InvitedUser>('/users', user, { idempotencyKey }),

	change: (userId: string, change: UserChange): Promise<UserRecord | null> =>
		client.patch<UserRecord>(path(userId), change),

	invite: (userId: string, idempotencyKey: string): Promise<Invitation | null> =>
		client.post<Invitation>(`${path(userId)}/invitations`, undefined, { idempotencyKey }),

	allowedPlugins: (userId: string): Promise<PluginRef[] | null> =>
		client
			.get<Page<PluginRef>>(`${path(userId)}/allowed-plugins`)
			.then((page) => page?.items ?? null),

	allowPlugin: (
		userId: string,
		pluginId: string,
		idempotencyKey: string
	): Promise<PluginRef | null> =>
		client.post<PluginRef>(
			`${path(userId)}/allowed-plugins`,
			{ plugin_id: pluginId },
			{ idempotencyKey }
		),

	disallowPlugin: (userId: string, pluginId: string): Promise<void> =>
		client
			.del<void>(`${path(userId)}/allowed-plugins/${encodeURIComponent(pluginId)}`)
			.then(() => undefined)
};
