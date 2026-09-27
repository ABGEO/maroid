export interface Environment {
  id: string;
  name: string;
  created_at: string;
  updated_at: string;
}

export interface EnvironmentPayload {
  name: string;
}

export interface Plant {
  id: string;
  name: string;
  species?: string;
  environment_id: string;
  created_at: string;
  updated_at: string;
}

export interface PlantPayload {
  name: string;
  species?: string;
  environment_id: string;
}
