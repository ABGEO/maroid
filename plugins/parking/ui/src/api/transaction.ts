import type { ApiClient, Page } from '@maroid/plugin-sdk';

import type { ChargeMonth, Transaction, TransactionFilter, TransactionTotal } from './types';

export function createTransactionApi(api: ApiClient) {
	return {
		/** Reads the first page of the filter, or the page that a link of an earlier page names. */
		page: (filter: TransactionFilter, link?: string) =>
			link
				? api.follow<Page<Transaction>>(link)
				: api.get<Page<Transaction>>('/transactions', { params: { ...filter } }),

		/** Reads one transaction, to learn whether Maroid holds any. */
		async any(): Promise<boolean | null> {
			const answered = await api.get<Page<Transaction>>('/transactions', {
				params: { limit: 1 }
			});

			return answered === null ? null : answered.items.length > 0;
		},

		total: (filter: TransactionFilter) =>
			api.get<TransactionTotal>('/transactions/total', { params: { ...filter } }),

		/** Reads each month from a date YYYY-MM-DD, or every month with no date. */
		async chargeMonths(from?: string): Promise<ChargeMonth[] | null> {
			const answered = await api.get<Page<ChargeMonth>>('/charge-months', { params: { from } });

			return answered?.items ?? null;
		}
	};
}
