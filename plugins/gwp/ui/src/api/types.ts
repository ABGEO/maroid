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

export interface Reading {
  id: string;
  serial_number: string;
  /** The date of the reading, YYYY-MM-DD. */
  reading_date: string;
  /** The value of the meter, in cubic metres. */
  value: number;
  /** The usage since the reading before it. Absent for the earliest reading of a meter. */
  usage?: number;
}
