import type { ApiClient, Page } from '@maroid/plugin-sdk';

import type { MoneyEntry, MonthTotal } from './types';

export function createMoneyApi(api: ApiClient) {
	return {
		/** Reads the first page, or the page that a link of an earlier page names. */
		page: (link?: string) =>
			link ? api.follow<Page<MoneyEntry>>(link) : api.get<Page<MoneyEntry>>('/money-entries'),

		/** Reads the totals of each month from from, a month YYYY-MM, to the current month. */
		monthTotals: (from: string) =>
			api.get<Page<MonthTotal>>('/monthly-money-totals', { params: { from } })
	};
}
