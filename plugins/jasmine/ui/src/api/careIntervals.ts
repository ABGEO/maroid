import type { ApiClient, Page } from '@maroid/plugin-sdk';

import type {
	CareInterval,
	CareIntervalFilter,
	CareIntervalPayload,
	MaintenanceType
} from './types';

export function createCareIntervalsApi(api: ApiClient) {
	return {
		list: (filter: CareIntervalFilter = {}, link?: string) =>
			link
				? api.follow<Page<CareInterval>>(link)
				: api.get<Page<CareInterval>>('/care-intervals', {
						params: {
							state: filter.state?.join(','),
							environment_id: filter.environment_id
						}
					}),
		listForPlant: (plantId: string) =>
			api.get<Page<CareInterval>>(`/plants/${plantId}/care-intervals`),
		put: (plantId: string, type: MaintenanceType, payload: CareIntervalPayload) =>
			api.put<CareInterval>(`/plants/${plantId}/care-intervals/${type}`, payload),
		remove: (plantId: string, type: MaintenanceType) =>
			api.del<null>(`/plants/${plantId}/care-intervals/${type}`)
	};
}
