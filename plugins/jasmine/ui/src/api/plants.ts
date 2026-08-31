import type { ApiClient } from '@maroid/plugin-sdk';

import type { Plant, PlantPayload } from './types';

export function createPlantsApi(api: ApiClient) {
  return {
    list: () => api.get<Plant[]>('/plants'),
    get: (id: string) => api.get<Plant>(`/plants/${id}`),
    create: (payload: PlantPayload) => api.post<Plant>('/plants', payload),
    update: (id: string, payload: PlantPayload) => api.put<Plant>(`/plants/${id}`, payload),
    remove: (id: string) => api.del<null>(`/plants/${id}`)
  };
}
