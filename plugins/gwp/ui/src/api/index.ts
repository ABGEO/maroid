import type { PluginHost } from '@maroid/plugin-sdk';

import { createConnectionApi } from './connection';
import { createReadingApi } from './reading';

export type { Connection, ConnectionState, Customer, Reading } from './types';

export function createGwpApi(host: PluginHost) {
	return {
		connection: createConnectionApi(host.api),
		readings: createReadingApi(host.api)
	};
}

export type GwpApi = ReturnType<typeof createGwpApi>;
