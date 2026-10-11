import type { ApiClient, Page, RequestOptions } from "@maroid/plugin-sdk";

import type {
  MaintenanceEntry,
  MaintenanceEntryBatchPayload,
  MaintenanceEntryFilter,
  MaintenanceEntryPayload,
} from "./types";

function listPath(filter: MaintenanceEntryFilter): string {
  const query = new URLSearchParams();
  for (const [name, value] of Object.entries(filter)) {
    if (value) {
      query.set(name, value);
    }
  }

  const encoded = query.toString();

  return encoded ? `/maintenance-entries?${encoded}` : "/maintenance-entries";
}

export function createMaintenanceEntriesApi(api: ApiClient) {
  return {
    list: (filter: MaintenanceEntryFilter = {}, link?: string) =>
      link
        ? api.follow<Page<MaintenanceEntry>>(link)
        : api.get<Page<MaintenanceEntry>>(listPath(filter)),
    get: (id: string) => api.get<MaintenanceEntry>(`/maintenance-entries/${id}`),
    create: (payload: MaintenanceEntryBatchPayload, options?: RequestOptions) =>
      api.post<{ items: MaintenanceEntry[] }>(
        "/maintenance-entries",
        payload,
        options,
      ),
    update: (id: string, payload: MaintenanceEntryPayload) =>
      api.put<MaintenanceEntry>(`/maintenance-entries/${id}`, payload),
    remove: (id: string) => api.del<null>(`/maintenance-entries/${id}`),
  };
}
