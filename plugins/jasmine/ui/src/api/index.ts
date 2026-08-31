import type { PluginHost } from '@maroid/plugin-sdk';

import { createEnvironmentsApi } from './environments';
import { createPlantsApi } from './plants';

export type { Environment, EnvironmentPayload, Plant, PlantPayload } from './types';

export function createJasmineApi(host: PluginHost) {
  return {
    environments: createEnvironmentsApi(host.api),
    plants: createPlantsApi(host.api)
  };
}
