import type { PluginHost } from '@maroid/plugin-sdk';

import { createConnectionApi } from './connection';
import { createTransactionApi } from './transaction';

export type { Connection, ConnectionState, Customer, Money, Transaction } from './types';

export function createGasApi(host: PluginHost) {
	return {
		connection: createConnectionApi(host.api),
		transactions: createTransactionApi(host.api)
	};
}

export type GasApi = ReturnType<typeof createGasApi>;
