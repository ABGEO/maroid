import type { ApiClient, Tagged } from '@maroid/plugin-sdk';

import type { Connection } from './types';

const CONNECTION = '/connections/self';

export function createConnectionApi(api: ApiClient) {
	return {
		/** Reads the connection, and the entity tag of the assignment when it holds one. */
		read: (): Promise<Tagged<Connection> | null> => api.getTagged<Connection>(CONNECTION),
		connect: () => api.put<Connection>(CONNECTION),
		disconnect: () => api.del<null>(CONNECTION),
		assign: (subscriberNumber: string, ifMatch?: string) =>
			api.put<Connection>(
				'/subscriber-assignments/self',
				{ subscriber_number: subscriberNumber },
				{ ifMatch }
			)
	};
}
