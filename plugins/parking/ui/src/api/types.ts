export type ConnectionState = "not_connected" | "connected";

/** An amount with its ISO 4217 currency. */
export interface Money {
  amount: number;
  currency: string;
}

export interface Account {
  first_name: string;
  last_name: string;
  balance: Money;
  free_parkings_left: number;
}

export interface PendingConnection {
  /** The masked phone number that received the code. */
  phone: string;
  resend_at: string;
  expires_at: string;
}

export interface Connection {
  state: ConnectionState;
  credentials_stored: boolean;
  /** Absent when the read of the account failed right after a verification. */
  account?: Account;
  pending?: PendingConnection;
}
