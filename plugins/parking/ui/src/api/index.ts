import type { PluginHost } from '@maroid/plugin-sdk';

import { createConnectionApi } from './connection';
import { createSessionApi } from './session';
import { createTransactionApi } from './transaction';

export type {
	Account,
	ChargeMonth,
	Connection,
	ConnectionState,
	LotRef,
	LotUsage,
	Money,
	PendingConnection,
	Session,
	SessionFilter,
	SessionTotal,
	Transaction,
	TransactionFilter,
	TransactionKind,
	TransactionTotal
} from './types';

export function createParkingApi(host: PluginHost) {
	return {
		connection: createConnectionApi(host.api),
		sessions: createSessionApi(host.api),
		transactions: createTransactionApi(host.api)
	};
}

export type ParkingApi = ReturnType<typeof createParkingApi>;
