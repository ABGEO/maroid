import type { ApiClient, Page } from '@maroid/plugin-sdk';

import { PAGE_LIMIT, readAll } from '../lib/pages';
import type { PortfolioShare, UnitPrice } from './types';

export function createPortfolioApi(api: ApiClient) {
	return {
		async shares(): Promise<PortfolioShare[] | null> {
			const answered = await api.get<Page<PortfolioShare>>('/portfolio-shares');

			return answered?.items ?? null;
		},

		/** Reads every page of the weekly unit prices from a date YYYY-MM-DD, or every week with no date. */
		unitPrices: (from?: string): Promise<UnitPrice[] | null> =>
			readAll((link) =>
				link
					? api.follow<Page<UnitPrice>>(link)
					: api.get<Page<UnitPrice>>('/portfolio-unit-prices', {
							params: { from, limit: PAGE_LIMIT }
						})
			)
	};
}
