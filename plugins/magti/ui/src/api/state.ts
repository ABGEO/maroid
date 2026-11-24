import type { ApiClient } from '@maroid/plugin-sdk';

import type { SubscriberState } from './types';

export function createStateApi(api: ApiClient) {
	return {
		read: () => api.get<SubscriberState>('/subscriber-states/self')
	};
}
