import type { ApiClient, Page } from "@maroid/plugin-sdk";

import type {
  CareInterval,
  CareIntervalFilter,
  CareIntervalPayload,
  MaintenanceType,
} from "./types";

function listPath(filter: CareIntervalFilter): string {
  const query = new URLSearchParams();
  if (filter.state?.length) {
    query.set("state", filter.state.join(","));
  }
  if (filter.environment_id) {
    query.set("environment_id", filter.environment_id);
  }

  const encoded = query.toString();

  return encoded ? `/care-intervals?${encoded}` : "/care-intervals";
}

export function createCareIntervalsApi(api: ApiClient) {
  return {
    list: (filter: CareIntervalFilter = {}, link?: string) =>
      link
        ? api.follow<Page<CareInterval>>(link)
        : api.get<Page<CareInterval>>(listPath(filter)),
    listForPlant: (plantId: string) =>
      api.get<Page<CareInterval>>(`/plants/${plantId}/care-intervals`),
    put: (
      plantId: string,
      type: MaintenanceType,
      payload: CareIntervalPayload,
    ) =>
      api.put<CareInterval>(
        `/plants/${plantId}/care-intervals/${type}`,
        payload,
      ),
    remove: (plantId: string, type: MaintenanceType) =>
      api.del<null>(`/plants/${plantId}/care-intervals/${type}`),
  };
}
