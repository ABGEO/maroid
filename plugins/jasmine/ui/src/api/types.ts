export interface EnvironmentReference {
  id: string;
  display_id: string;
  name: string;
}

export interface Environment extends EnvironmentReference {
  active_plant_count: number;
  created_at: string;
  updated_at: string;
}

export interface EnvironmentPayload {
  name: string;
}

export type PlantStatus = "active" | "inactive";

export interface PlantReference {
  id: string;
  display_id: string;
  common_name: string;
  environment: EnvironmentReference;
}

export interface Plant extends PlantReference {
  scientific_name?: string;
  acquired_on?: string;
  care_notes?: string;
  status: PlantStatus;
  created_at: string;
  updated_at: string;
}

export interface PlantPayload {
  common_name: string;
  scientific_name?: string | null;
  environment_id: string;
  acquired_on?: string | null;
  care_notes?: string | null;
}

export interface PlantUpdatePayload extends PlantPayload {
  status: PlantStatus;
}

export type MaintenanceType =
  | "watering"
  | "fertilizing"
  | "pruning"
  | "repotting"
  | "pest_treatment"
  | "cleaning"
  | "misting"
  | "other";

export interface MaintenanceEntry {
  id: string;
  plant: PlantReference;
  type: MaintenanceType;
  performed_at: string;
  note?: string;
  created_at: string;
  updated_at: string;
}

export interface MaintenanceEntryBatchPayload {
  plant_ids: string[];
  type: MaintenanceType;
  performed_at?: string;
  note?: string | null;
}

export interface MaintenanceEntryPayload {
  type: MaintenanceType;
  performed_at: string;
  note?: string | null;
}

export interface MaintenanceEntryFilter {
  plant_id?: string;
  type?: MaintenanceType;
  sort?: "performed_at" | "-performed_at";
}

export type CareState = "on_schedule" | "due" | "overdue";

export interface CareInterval {
  id: string;
  plant: PlantReference;
  type: MaintenanceType;
  days: number;
  set_at: string;
  last_performed_at?: string;
  counted_from: string;
  days_since: number;
  state?: CareState;
  updated_at: string;
}

export interface CareIntervalPayload {
  days: number;
}

export interface CareIntervalFilter {
  state?: CareState[];
  environment_id?: string;
}
