import type { ApiClient, Page } from '@maroid/plugin-sdk';

import type { Transaction } from './types';

/** The most transactions that one page answers, which the chart reads in few pages. */
const SERIES_PAGE_LIMIT = 100;

export function createTransactionApi(api: ApiClient) {
	return {
		/** Reads the first page, or the page that a link of an earlier page names. */
		page: (link?: string) =>
			link ? api.follow<Page<Transaction>>(link) : api.get<Page<Transaction>>('/transactions'),

		/**
		 * Reads every transaction dated on or after from, a date YYYY-MM-DD, across the
		 * pages. It answers null when the client sent the person to sign in.
		 */
		async since(from: string): Promise<Transaction[] | null> {
			const transactions: Transaction[] = [];

			let answered = await api.get<Page<Transaction>>('/transactions', {
				params: { from, limit: SERIES_PAGE_LIMIT }
			});

			while (answered !== null) {
				transactions.push(...answered.items);

				if (answered.next === undefined) {
					return transactions;
				}

				answered = await api.follow<Page<Transaction>>(answered.next);
			}

			return null;
		}
	};
}
