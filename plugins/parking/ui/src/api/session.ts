import type { ApiClient, Page } from '@maroid/plugin-sdk';

import type { LotUsage, Plate, Session, SessionFilter, SessionTotal } from './types';

export function createSessionApi(api: ApiClient) {
	return {
		/** Reads the first page of the filter, or the page that a link of an earlier page names. */
		page: (filter: SessionFilter, link?: string) =>
			link
				? api.follow<Page<Session>>(link)
				: api.get<Page<Session>>('/sessions', { params: { ...filter } }),

		/** Reads one session, to learn whether Maroid holds any. */
		async any(): Promise<boolean | null> {
			const answered = await api.get<Page<Session>>('/sessions', { params: { limit: 1 } });

			return answered === null ? null : answered.items.length > 0;
		},

		total: (filter: SessionFilter) =>
			api.get<SessionTotal>('/sessions/total', { params: { ...filter } }),

		async plates(): Promise<string[] | null> {
			const answered = await api.get<Page<Plate>>('/plates');

			return answered?.items.map((item) => item.plate) ?? null;
		},

		/** Reads the 5 busiest parking lots from a date YYYY-MM-DD, or of every session. */
		async lotUsages(from?: string): Promise<LotUsage[] | null> {
			const answered = await api.get<Page<LotUsage>>('/lot-usages', { params: { from } });

			return answered?.items ?? null;
		}
	};
}
