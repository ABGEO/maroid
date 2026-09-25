import type { ApiClient, Page } from "@maroid/plugin-sdk";

import type { Plant, PlantPayload } from "./types";

export function createPlantsApi(api: ApiClient) {
  return {
    // The collection answers a page. A null means the client is redirecting to
    // a sign in, so the caller still tells it from an empty collection.
    list: () =>
      api.get<Page<Plant>>("/plants").then((page) => page?.items ?? null),
    get: (id: string) => api.get<Plant>(`/plants/${id}`),
    create: (payload: PlantPayload) => api.post<Plant>("/plants", payload),
    update: (id: string, payload: PlantPayload) =>
      api.put<Plant>(`/plants/${id}`, payload),
    remove: (id: string) => api.del<null>(`/plants/${id}`),
  };
}
