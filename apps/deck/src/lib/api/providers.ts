import type { Page, Tagged } from '@maroid/api-client';

import { client } from './client';
import type { Provider, ProviderPreset } from './types';

function path(providerId: string): string {
	return `/providers/${encodeURIComponent(providerId)}`;
}

export interface NewProvider {
	preset: ProviderPreset;
	id?: string;
	name?: string;
	issuer?: string;
	client_id?: string;
	client_secret?: string;
	user_id_key?: string;
	scopes?: string[];
	options?: Record<string, unknown>;
}

/** A merge patch. A null option removes that option. */
export interface ProviderChange {
	name?: string;
	client_id?: string;
	client_secret?: string;
	scopes?: string[];
	options?: Record<string, unknown>;
}

export const providers = {
	list: (): Promise<Provider[] | null> =>
		client.get<Page<Provider>>('/providers').then((page) => page?.items ?? null),

	get: (providerId: string): Promise<Tagged<Provider> | null> =>
		client.getTagged<Provider>(path(providerId)),

	create: (provider: NewProvider, idempotencyKey: string): Promise<Provider | null> =>
		client.post<Provider>('/providers', provider, { idempotencyKey }),

	change: (providerId: string, change: ProviderChange, etag?: string): Promise<Provider | null> =>
		client.patch<Provider>(path(providerId), change, { ifMatch: etag }),

	remove: (providerId: string): Promise<void> =>
		client.del<void>(path(providerId)).then(() => undefined)
};
