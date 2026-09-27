import type { ApiClient, Page, RequestOptions } from "@maroid/plugin-sdk";

import type { Plant, PlantPayload } from "./types";

export function createPlantsApi(api: ApiClient) {
  return {
    list: (link?: string) =>
      link
        ? api.follow<Page<Plant>>(link)
        : api.get<Page<Plant>>("/plants"),
    get: (id: string) => api.get<Plant>(`/plants/${id}`),
    create: (payload: PlantPayload, options?: RequestOptions) =>
      api.post<Plant>("/plants", payload, options),
    update: (id: string, payload: PlantPayload) =>
      api.put<Plant>(`/plants/${id}`, payload),
    remove: (id: string) => api.del<null>(`/plants/${id}`),
  };
}
