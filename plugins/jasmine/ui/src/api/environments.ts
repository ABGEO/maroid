import type { ApiClient, Page, RequestOptions } from "@maroid/plugin-sdk";

import type { Environment, EnvironmentPayload } from "./types";

export function createEnvironmentsApi(api: ApiClient) {
  return {
    list: (link?: string) =>
      link
        ? api.follow<Page<Environment>>(link)
        : api.get<Page<Environment>>("/environments"),
    get: (id: string) => api.get<Environment>(`/environments/${id}`),
    create: (payload: EnvironmentPayload, options?: RequestOptions) =>
      api.post<Environment>("/environments", payload, options),
    update: (id: string, payload: EnvironmentPayload) =>
      api.put<Environment>(`/environments/${id}`, payload),
    remove: (id: string) => api.del<null>(`/environments/${id}`),
  };
}
