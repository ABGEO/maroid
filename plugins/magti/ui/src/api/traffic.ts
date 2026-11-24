import type { ApiClient } from '@maroid/plugin-sdk';

import { readEvery, SERIES_PAGE_LIMIT } from './collect';
import type { TrafficDay } from './types';

export function createTrafficApi(api: ApiClient) {
	return {
		/** Reads every day from from to to, both dates YYYY-MM-DD, the oldest first. */
		between: (from: string, to: string) =>
			readEvery<TrafficDay>(api, '/traffic-days', { from, to, limit: SERIES_PAGE_LIMIT })
	};
}
