import type { Page } from '@maroid/api-client';

import { client } from './client';
import type { ProviderIdentity } from './types';
import { LOCAL_PROVIDER } from './users';

function path(provider: string): string {
	return `/auth/identities/${encodeURIComponent(provider)}`;
}

export const identities = {
	list: (): Promise<ProviderIdentity[] | null> =>
		client.get<Page<ProviderIdentity>>('/auth/identities').then((page) => page?.items ?? null),

	detach: (provider: string): Promise<void> =>
		client.del<void>(path(provider)).then(() => undefined),

	changePassword: (current: string, password: string): Promise<void> =>
		client
			.patch<void>(path(LOCAL_PROVIDER), { current_password: current, password })
			.then(() => undefined)
};
