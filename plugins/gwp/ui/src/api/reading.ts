import type { ApiClient, Page } from '@maroid/plugin-sdk';

import type { Reading } from './types';

/** The most readings that one page answers, which the chart reads in few pages. */
const SERIES_PAGE_LIMIT = 100;

export function createReadingApi(api: ApiClient) {
	return {
		/** Reads the first page, or the page that a link of an earlier page names. */
		page: (link?: string) =>
			link ? api.follow<Page<Reading>>(link) : api.get<Page<Reading>>('/readings'),

		/**
		 * Reads every reading dated on or after from, a date YYYY-MM-DD, across the
		 * pages. It answers null when the client sent the person to sign in.
		 */
		async since(from: string): Promise<Reading[] | null> {
			const readings: Reading[] = [];

			let answered = await api.get<Page<Reading>>('/readings', {
				params: { from, limit: SERIES_PAGE_LIMIT }
			});

			while (answered !== null) {
				readings.push(...answered.items);

				if (answered.next === undefined) {
					return readings;
				}

				answered = await api.follow<Page<Reading>>(answered.next);
			}

			return null;
		}
	};
}
