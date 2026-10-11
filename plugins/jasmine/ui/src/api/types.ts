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

export interface Plant {
  id: string;
  display_id: string;
  common_name: string;
  scientific_name?: string;
  environment: EnvironmentReference;
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
