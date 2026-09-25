import type { ApiClient, Page } from "@maroid/plugin-sdk";

import type { Environment, EnvironmentPayload } from "./types";

export function createEnvironmentsApi(api: ApiClient) {
  return {
    // The collection answers a page. A null means the client is redirecting to
    // a sign in, so the caller still tells it from an empty collection.
    list: () =>
      api
        .get<Page<Environment>>("/environments")
        .then((page) => page?.items ?? null),
    get: (id: string) => api.get<Environment>(`/environments/${id}`),
    create: (payload: EnvironmentPayload) =>
      api.post<Environment>("/environments", payload),
    update: (id: string, payload: EnvironmentPayload) =>
      api.put<Environment>(`/environments/${id}`, payload),
    remove: (id: string) => api.del<null>(`/environments/${id}`),
  };
}
