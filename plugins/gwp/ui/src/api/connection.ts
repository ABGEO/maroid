import type { ApiClient } from "@maroid/plugin-sdk";

import type { Connection } from "./types";

export function createConnectionApi(api: ApiClient) {
  return {
    read: () => api.get<Connection>("/connections/self"),
    connect: () => api.put<Connection>("/connections/self"),
    disconnect: () => api.del<null>("/connections/self"),
    assign: (customerNumber: string) =>
      api.put<Connection>("/customer-assignments/self", {
        customer_number: customerNumber,
      }),
  };
}
