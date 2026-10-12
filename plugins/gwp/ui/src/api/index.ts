import type { PluginHost } from "@maroid/plugin-sdk";

import { createConnectionApi } from "./connection";

export type { Connection, ConnectionState, Customer } from "./types";

export function createGwpApi(host: PluginHost) {
  return {
    connection: createConnectionApi(host.api),
  };
}

export type GwpApi = ReturnType<typeof createGwpApi>;
