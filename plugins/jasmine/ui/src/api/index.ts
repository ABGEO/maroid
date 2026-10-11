import type { PluginHost } from "@maroid/plugin-sdk";

import { createEnvironmentsApi } from "./environments";
import { createMaintenanceEntriesApi } from "./maintenanceEntries";
import { createPlantsApi } from "./plants";

export type {
  Environment,
  EnvironmentPayload,
  EnvironmentReference,
  MaintenanceEntry,
  MaintenanceEntryBatchPayload,
  MaintenanceEntryFilter,
  MaintenanceEntryPayload,
  MaintenanceType,
  Plant,
  PlantPayload,
  PlantReference,
  PlantStatus,
  PlantUpdatePayload,
} from "./types";

export function createJasmineApi(host: PluginHost) {
  return {
    environments: createEnvironmentsApi(host.api),
    plants: createPlantsApi(host.api),
    maintenanceEntries: createMaintenanceEntriesApi(host.api),
  };
}
