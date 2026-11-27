import type { ApiClient, Page } from '@maroid/plugin-sdk';

import type {
	Contribution,
	ContributionFilter,
	ContributionMonth,
	ContributionTotal,
	Employer
} from './types';

export function createContributionApi(api: ApiClient) {
	return {
		/** Reads the first page of the filter, or the page that a link of an earlier page names. */
		page: (filter: ContributionFilter, link?: string) =>
			link
				? api.follow<Page<Contribution>>(link)
				: api.get<Page<Contribution>>('/contributions', { params: { ...filter } }),

		total: (filter: ContributionFilter) =>
			api.get<ContributionTotal>('/contributions/total', { params: { ...filter } }),

		/** Reads the sum of each month from a date YYYY-MM-DD, or of every month with no date. */
		async months(from?: string): Promise<ContributionMonth[] | null> {
			const answered = await api.get<Page<ContributionMonth>>('/contribution-months', {
				params: { from }
			});

			return answered?.items ?? null;
		},

		async employers(): Promise<Employer[] | null> {
			const answered = await api.get<Page<Employer>>('/employers');

			return answered?.items ?? null;
		}
	};
}
