import type { PluginHost } from '@maroid/plugin-sdk';

import { createBillApi } from './bill';
import { createConnectionApi } from './connection';
import { createTransactionApi } from './transaction';

export type { Bill, Connection, ConnectionState, Customer, Money, Transaction } from './types';

export function createElectricityApi(host: PluginHost) {
	return {
		connection: createConnectionApi(host.api),
		transactions: createTransactionApi(host.api),
		bills: createBillApi(host.api)
	};
}

export type ElectricityApi = ReturnType<typeof createElectricityApi>;
