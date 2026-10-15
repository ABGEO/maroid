import type { ApiClient, Page, RequestOptions } from '@maroid/plugin-sdk';

import type {
	MaintenanceEntry,
	MaintenanceEntryBatchPayload,
	MaintenanceEntryFilter,
	MaintenanceEntryPayload
} from './types';

export function createMaintenanceEntriesApi(api: ApiClient) {
	return {
		list: (filter: MaintenanceEntryFilter = {}, link?: string) =>
			link
				? api.follow<Page<MaintenanceEntry>>(link)
				: api.get<Page<MaintenanceEntry>>('/maintenance-entries', {
						params: { ...filter }
					}),
		get: (id: string) => api.getTagged<MaintenanceEntry>(`/maintenance-entries/${id}`),
		create: (payload: MaintenanceEntryBatchPayload, options?: RequestOptions) =>
			api.post<{ items: MaintenanceEntry[] }>('/maintenance-entries', payload, options),
		update: (id: string, payload: MaintenanceEntryPayload, ifMatch?: string) =>
			api.put<MaintenanceEntry>(`/maintenance-entries/${id}`, payload, {
				ifMatch
			}),
		remove: (id: string, ifMatch?: string) =>
			api.del<null>(`/maintenance-entries/${id}`, { ifMatch })
	};
}
