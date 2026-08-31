import type { ApiClient } from '@maroid/plugin-sdk';

import type { Environment, EnvironmentPayload } from './types';

export function createEnvironmentsApi(api: ApiClient) {
  return {
    list: () => api.get<Environment[]>('/environments'),
    get: (id: string) => api.get<Environment>(`/environments/${id}`),
    create: (payload: EnvironmentPayload) => api.post<Environment>('/environments', payload),
    update: (id: string, payload: EnvironmentPayload) =>
      api.put<Environment>(`/environments/${id}`, payload),
    remove: (id: string) => api.del<null>(`/environments/${id}`)
  };
}
