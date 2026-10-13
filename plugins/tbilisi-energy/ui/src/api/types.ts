export type ConnectionState = "not_connected" | "connected";

export interface Customer {
  /** The number and the control number, joined by a hyphen. */
  customer_number: string;
  name: string;
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

export interface Transaction {
  id: string;
  /** The date of the transaction, YYYY-MM-DD. */
  transaction_date: string;
  /** The name of the operation, as Tbilisi Energy gives it. */
  operation: string;
  /** The amount, in lari. */
  amount: number;
  /** The balance after the transaction, in lari. */
  balance: number;
  /** The usage of a reading, in cubic metres. Absent for a payment. */
  usage?: number;
  /** The value of the meter, in cubic metres. Absent for a payment. */
  meter_value?: number;
  /** The bill at Tbilisi Energy. Absent when it holds none. */
  bill_url?: string;
  /** The meter photo at Tbilisi Energy. Absent when it holds none. */
  meter_photo_url?: string;
}
