import type { PluginHost } from "@maroid/plugin-sdk";

import { createConnectionApi } from "./connection";

export type {
  Account,
  Connection,
  ConnectionState,
  PendingConnection,
} from "./types";

export function createParkingApi(host: PluginHost) {
  return {
    connection: createConnectionApi(host.api),
  };
}

export type ParkingApi = ReturnType<typeof createParkingApi>;
