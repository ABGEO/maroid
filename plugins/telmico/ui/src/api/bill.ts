import type { ApiClient, Page } from '@maroid/plugin-sdk';

import type { Bill } from './types';

export function createBillApi(api: ApiClient) {
	return {
		/** Reads the first page, or the page that a link of an earlier page names. */
		page: (link?: string) => (link ? api.follow<Page<Bill>>(link) : api.get<Page<Bill>>('/bills'))
	};
}
