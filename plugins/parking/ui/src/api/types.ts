export type ConnectionState = 'not_connected' | 'connected';

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

/** A parking lot, named by its number and its address. */
export interface LotRef {
	lot_number: string;
	address: string;
}

/** A stored session. An amount of 0 is a free parking. */
export interface Session {
	id: string;
	lot_number: string;
	address: string;
	plate: string;
	started_at: string;
	ended_at: string;
	duration_seconds: number;
	amount: Money;
	free: boolean;
}

/** The filter of the sessions page. Each date is YYYY-MM-DD, both included. */
export interface SessionFilter {
	from?: string;
	to?: string;
	plate?: string;
	free?: 'true' | 'false';
}

export interface SessionTotal {
	count: number;
	duration_seconds: number;
	amount: Money;
}

export interface Plate {
	plate: string;
}

/** The sessions of one parking lot. The address is the one of its latest session. */
export interface LotUsage {
	lot_number: string;
	address: string;
	count: number;
	amount: Money;
}

export type TransactionKind = 'top_up' | 'parking_payment' | 'permit_purchase' | 'other';

/** A stored transaction. The amount carries no sign: the kind gives the direction. */
export interface Transaction {
	id: string;
	occurred_at: string;
	kind: TransactionKind;
	description: string;
	amount: Money;
	balance_after: Money;
	/** The parking lot of the paid session, when Maroid holds that session. */
	session?: LotRef;
}

/** The filter of the transactions page. Each date is YYYY-MM-DD, both included. */
export interface TransactionFilter {
	from?: string;
	to?: string;
	kind?: TransactionKind;
}

export interface TransactionTotal {
	count: number;
	top_ups: Money;
	charges: Money;
}

/** One month YYYY-MM: the sum of its charges, and the balance after its latest transaction. */
export interface ChargeMonth {
	month: string;
	charges: Money;
	balance: Money;
}
