export type ConnectionState = 'not_connected' | 'connected';

export type SubscriberKind = 'mobile' | 'home_line' | 'other';

export interface Subscriber {
	/** The number of the subscriber at Magti. */
	subscriber_number: string;
	/** The label of the subscriber, as the portal gives it. */
	label: string;
	kind: SubscriberKind;
	/** Whether the subscriber is assigned to the acting workspace. */
	assigned: boolean;
}

export interface Connection {
	state: ConnectionState;
	credentials_stored: boolean;
	/** Present when the workspace is connected. */
	subscribers?: Subscriber[];
}

/** An amount of money, the money object of Zalando rule 173. */
export interface Money {
	amount: number;
	/** The ISO 4217 code. Magti bills in lari alone. */
	currency: string;
}

export interface Plan {
	/** The name of the plan. A home line answers none. */
	name?: string;
	price: Money;
	/** The date of the renewal in Tbilisi, YYYY-MM-DD. A home line answers none. */
	renews_on?: string;
	days_left: number;
}

/** What remains of the plan of a mobile subscriber. */
export interface Allowances {
	data_mb: number;
	call_seconds: number;
	sms: number;
}

/** The state of the assigned subscriber, read from the portal at each read. */
export interface SubscriberState {
	subscriber_number: string;
	kind: SubscriberKind;
	/** The status, as the portal gives it. */
	status: string;
	balance: Money;
	/** Absent when the subscriber holds no active plan. */
	plan?: Plan;
	/** A mobile subscriber alone answers them. */
	allowances?: Allowances;
	/** The speed of a home line, as the portal gives it. */
	speed?: string;
}

/** The traffic of one day. A value that the portal gave no row for is absent. */
export interface TrafficDay {
	/** The day, YYYY-MM-DD. */
	day: string;
	data_mb?: number;
	call_in_seconds?: number;
	call_out_seconds?: number;
	sms_sent?: number;
	sms_received?: number;
}

export type Direction = 'in' | 'out';

export interface MoneyEntry {
	id: string;
	/** The moment, RFC 3339 in UTC. */
	occurred_at: string;
	/** A top up is `in`, and a charge is `out`. */
	direction: Direction;
	/** The kind of the entry, as the portal gives it. */
	kind: string;
	/** The amount, never below zero. */
	amount: Money;
}

export interface MonthTotal {
	/** The month in Tbilisi, YYYY-MM. */
	month: string;
	top_ups: Money;
	charges: Money;
}
