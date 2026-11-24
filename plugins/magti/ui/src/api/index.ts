import type { PluginHost } from '@maroid/plugin-sdk';

import { createConnectionApi } from './connection';
import { createMoneyApi } from './money';
import { createStateApi } from './state';
import { createTrafficApi } from './traffic';

export type {
	Allowances,
	Connection,
	ConnectionState,
	Direction,
	Money,
	MoneyEntry,
	MonthTotal,
	Plan,
	Subscriber,
	SubscriberKind,
	SubscriberState,
	TrafficDay
} from './types';

export function createTelecomApi(host: PluginHost) {
	return {
		connection: createConnectionApi(host.api),
		state: createStateApi(host.api),
		traffic: createTrafficApi(host.api),
		money: createMoneyApi(host.api)
	};
}

export type TelecomApi = ReturnType<typeof createTelecomApi>;
