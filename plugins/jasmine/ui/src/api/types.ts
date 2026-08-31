export interface Environment {
  id: string;
  name: string;
  createdAt: string;
  updatedAt: string;
}

export interface EnvironmentPayload {
  name: string;
}

export interface Plant {
  id: string;
  name: string;
  species?: string;
  environmentId: string;
  createdAt: string;
  updatedAt: string;
}

export interface PlantPayload {
  name: string;
  species?: string;
  environmentId: string;
}
