import type { Page } from '@maroid/api-client';

import { client } from './client';
import type {
	NotificationProvider,
	NotificationProviderKind,
	NotificationRoute,
	NotificationTopic,
	SettingsInput
} from './types';

function base(workspaceId: string): string {
	return `/workspaces/${encodeURIComponent(workspaceId)}`;
}

function providerPath(workspaceId: string, providerId: string): string {
	return `${base(workspaceId)}/notification-providers/${encodeURIComponent(providerId)}`;
}

function routePath(workspaceId: string, routeId: string): string {
	return `${base(workspaceId)}/notification-routes/${encodeURIComponent(routeId)}`;
}

export interface NewNotificationProvider {
	kind: string;
	name: string;
	fields: SettingsInput;
	enabled?: boolean;
}

/** A merge patch. A secret that holds the mask keeps its stored value. */
export interface NotificationProviderChange {
	name?: string;
	fields?: SettingsInput;
	enabled?: boolean;
}

export interface NewNotificationRoute {
	plugin_id: string;
	topic_id: string;
	provider_id: string;
	destination: SettingsInput;
	enabled?: boolean;
}

/** A merge patch. A null member of the destination removes that field. */
export interface NotificationRouteChange {
	destination?: SettingsInput;
	enabled?: boolean;
}

export const notifications = {
	kinds: (workspaceId: string): Promise<NotificationProviderKind[] | null> =>
		client
			.get<Page<NotificationProviderKind>>(`${base(workspaceId)}/notification-provider-kinds`)
			.then((page) => page?.items ?? null),

	providers: (workspaceId: string): Promise<NotificationProvider[] | null> =>
		client
			.get<Page<NotificationProvider>>(`${base(workspaceId)}/notification-providers`)
			.then((page) => page?.items ?? null),

	createProvider: (
		workspaceId: string,
		provider: NewNotificationProvider,
		idempotencyKey: string
	): Promise<NotificationProvider | null> =>
		client.post<NotificationProvider>(`${base(workspaceId)}/notification-providers`, provider, {
			idempotencyKey
		}),

	changeProvider: (
		workspaceId: string,
		providerId: string,
		change: NotificationProviderChange
	): Promise<NotificationProvider | null> =>
		client.patch<NotificationProvider>(providerPath(workspaceId, providerId), change),

	removeProvider: (workspaceId: string, providerId: string): Promise<void> =>
		client.del<void>(providerPath(workspaceId, providerId)).then(() => undefined),

	topics: (workspaceId: string): Promise<NotificationTopic[] | null> =>
		client
			.get<Page<NotificationTopic>>(`${base(workspaceId)}/notification-topics`)
			.then((page) => page?.items ?? null),

	createRoute: (
		workspaceId: string,
		route: NewNotificationRoute,
		idempotencyKey: string
	): Promise<NotificationRoute | null> =>
		client.post<NotificationRoute>(`${base(workspaceId)}/notification-routes`, route, {
			idempotencyKey
		}),

	changeRoute: (
		workspaceId: string,
		routeId: string,
		change: NotificationRouteChange
	): Promise<NotificationRoute | null> =>
		client.patch<NotificationRoute>(routePath(workspaceId, routeId), change),

	removeRoute: (workspaceId: string, routeId: string): Promise<void> =>
		client.del<void>(routePath(workspaceId, routeId)).then(() => undefined),

	testRoute: (workspaceId: string, routeId: string): Promise<void> =>
		client.post<void>(`${routePath(workspaceId, routeId)}/test-messages`).then(() => undefined)
};
