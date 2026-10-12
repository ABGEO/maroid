export type ConnectionState = "not_connected" | "connected";

export interface Customer {
  customer_number: string;
  /** The number on the communal bill. Absent when GWP gives none. */
  communal_number?: string;
  address: string;
  /** Whether the customer is assigned to the acting workspace. */
  assigned: boolean;
}

export interface Connection {
  state: ConnectionState;
  credentials_stored: boolean;
  /** Present when the workspace is connected. */
  customers?: Customer[];
}
