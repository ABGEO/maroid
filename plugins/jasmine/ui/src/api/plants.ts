import type { ApiClient, Page, RequestOptions } from '@maroid/plugin-sdk';

import type { Plant, PlantFilter, PlantPayload, PlantUpdatePayload } from './types';

export function createPlantsApi(api: ApiClient) {
	return {
		list: (filter: PlantFilter = {}, link?: string) =>
			link
				? api.follow<Page<Plant>>(link)
				: api.get<Page<Plant>>('/plants', { params: { ...filter } }),
		get: (id: string) => api.getTagged<Plant>(`/plants/${id}`),
		create: (payload: PlantPayload, options?: RequestOptions) =>
			api.post<Plant>('/plants', payload, options),
		update: (id: string, payload: PlantUpdatePayload, ifMatch?: string) =>
			api.put<Plant>(`/plants/${id}`, payload, { ifMatch }),
		remove: (id: string, ifMatch?: string) => api.del<null>(`/plants/${id}`, { ifMatch })
	};
}
