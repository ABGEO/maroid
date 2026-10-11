export type ConnectionState = "not_connected" | "connected";

export interface Account {
  first_name: string;
  last_name: string;
  /** The balance in GEL. */
  balance: number;
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
  account?: Account;
  pending?: PendingConnection;
}
