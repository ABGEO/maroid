import type { ApiClient, Page, RequestOptions } from '@maroid/plugin-sdk';

import type { Moment, MomentInput, MomentUpdate } from './types';

export function createMomentsApi(api: ApiClient) {
	const momentsOf = (plantId: string) => `/plants/${plantId}/moments`;

	return {
		list: (plantId: string, limit: number, link?: string) =>
			link
				? api.follow<Page<Moment>>(link)
				: api.get<Page<Moment>>(momentsOf(plantId), { params: { limit } }),
		create: (plantId: string, payload: MomentInput, options?: RequestOptions) =>
			api.post<Moment>(momentsOf(plantId), payload, options),
		update: (plantId: string, id: string, payload: MomentUpdate) =>
			api.put<Moment>(`${momentsOf(plantId)}/${id}`, payload),
		remove: (plantId: string, id: string) => api.del<null>(`${momentsOf(plantId)}/${id}`),
		addPhotos: (plantId: string, id: string, fileIds: string[]) =>
			api.post<Moment>(`${momentsOf(plantId)}/${id}/photos`, { file_ids: fileIds }),
		removePhoto: (plantId: string, momentId: string, id: string) =>
			api.del<null>(`${momentsOf(plantId)}/${momentId}/photos/${id}`)
	};
}
