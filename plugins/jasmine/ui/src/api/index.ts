import type { PluginHost } from "@maroid/plugin-sdk";

import { createCareIntervalsApi } from "./careIntervals";
import { createEnvironmentsApi } from "./environments";
import { createMaintenanceEntriesApi } from "./maintenanceEntries";
import { createPlantsApi } from "./plants";

export type {
  CareInterval,
  CareIntervalFilter,
  CareIntervalPayload,
  CareState,
  Environment,
  EnvironmentPayload,
  EnvironmentReference,
  MaintenanceEntry,
  MaintenanceEntryBatchPayload,
  MaintenanceEntryFilter,
  MaintenanceEntryPayload,
  MaintenanceType,
  Plant,
  PlantFilter,
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
    careIntervals: createCareIntervalsApi(host.api),
  };
}

export type JasmineApi = ReturnType<typeof createJasmineApi>;
