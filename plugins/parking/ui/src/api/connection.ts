import type { ApiClient } from '@maroid/plugin-sdk';

import type { Connection, PendingConnection } from './types';

export function createConnectionApi(api: ApiClient) {
	return {
		read: () => api.get<Connection>('/connections/self'),
		disconnect: () => api.del<null>('/connections/self'),
		start: () => api.post<PendingConnection>('/pending-connections'),
		requestCode: () => api.post<PendingConnection>('/pending-connections/self/code-requests'),
		verify: (code: string) =>
			api.post<Connection>('/pending-connections/self/verifications', { code })
	};
}
