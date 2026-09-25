import type { Page } from '@maroid/api-client';

import { client } from './client';
import type { ProviderIdentity } from './types';

function path(provider: string): string {
	return `/auth/identities/${encodeURIComponent(provider)}`;
}

export const identities = {
	list: (): Promise<ProviderIdentity[] | null> =>
		client.get<Page<ProviderIdentity>>('/auth/identities').then((page) => page?.items ?? null),

	detach: (provider: string): Promise<void> =>
		client.del<void>(path(provider)).then(() => undefined)
};
