import type { ApiClient } from '@maroid/plugin-sdk';

import type { SummaryView } from './types';

export function createSummaryApi(api: ApiClient) {
	return {
		read: () => api.get<SummaryView>('/account-summaries/self')
	};
}
