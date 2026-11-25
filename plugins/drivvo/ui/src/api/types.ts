export type ConnectionState = 'not_connected' | 'connected';

/** A vehicle as the car log service lists it at the connection. */
export interface ServiceVehicle {
	name: string;
	make: string;
	model: string;
	active: boolean;
}

export interface Connection {
	state: ConnectionState;
	credentials_stored: boolean;
	/** Present when the workspace is connected. */
	vehicles?: ServiceVehicle[];
}

/** An amount of money, the money object of Zalando rule 173. */
export interface Money {
	amount: number;
	/** The ISO 4217 code of the car log settings. */
	currency: string;
}

export type DistanceUnit = 'km' | 'mi';
export type VolumeUnit = 'l' | 'gal_us' | 'gal_imp';
export type ConsumptionShape = 'distance_per_volume' | 'volume_per_100';

export interface Units {
	distance: DistanceUnit;
	volume: VolumeUnit;
	consumption_shape: ConsumptionShape;
}

export interface Vehicle {
	id: string;
	name: string;
	make: string;
	model: string;
	year?: number;
	active: boolean;
	principal: boolean;
	/** The one vehicle that a page with no vehicle opens. */
	default: boolean;
	units: Units;
}

export type Kind = 'refuel' | 'maintenance' | 'expense' | 'income' | 'odometer';

/** A period of days, YYYY-MM-DD. An absent end leaves that end open. */
export interface Period {
	from?: string;
	to?: string;
}

export interface MonthFigures {
	/** The month, YYYY-MM. */
	month: string;
	fuel: Money;
	maintenance: Money;
	expense: Money;
	income: Money;
}

export interface TypeSpend {
	kind: 'maintenance' | 'expense';
	name: string;
	amount: Money;
}

export interface Overview {
	from?: string;
	to?: string;
	spend: { fuel: Money; maintenance: Money; expense: Money; total: Money };
	income: Money;
	months: MonthFigures[];
	types: TypeSpend[];
	distance: number | null;
	cost_per_distance: { fuel: Money | null; total: Money | null };
	consumption: { value: number | null; shape: ConsumptionShape };
}

export interface LogEntryItem {
	id: string;
	kind: Kind;
	/** The moment, RFC 3339 in UTC. */
	occurred_at: string;
	odometer: number;
	/** Absent for an odometer entry. */
	amount?: Money;
	title?: string;
	/** The names of the fuels or of the types, in the order of the lines. */
	labels: string[];
	volume?: number;
	unit_price?: Money;
	full_tank?: boolean;
	/** The consumption of the part that the refuel ends. */
	consumption?: number;
	place?: string;
	income_type?: string;
}

export interface LogEntryTotals {
	kind: Kind;
	count: number;
	amount: Money;
	/** A refuel alone answers it. */
	volume?: number;
}

export interface Line {
	position: number;
	item?: string;
	amount: Money;
	unit_price?: Money;
	volume?: number;
	full_tank?: boolean;
	free?: boolean;
}

export interface Instalment {
	number: number;
	/** The day, YYYY-MM-DD. */
	due_on: string;
	amount: Money;
}

/** One log entry. A member that the kind does not hold is absent. */
export interface LogEntry {
	id: string;
	kind: Kind;
	occurred_at: string;
	odometer: number;
	note?: string;
	amount?: Money;
	title?: string;
	station?: string;
	place?: string;
	payment_method?: string;
	reason?: string;
	income_type?: string;
	missed_refuel?: boolean;
	battery_start_pct?: number;
	battery_end_pct?: number;
	charge_minutes?: number;
	lines?: Line[];
	instalments?: Instalment[];
}

export interface Reminder {
	id: string;
	kind: 'maintenance' | 'expense';
	type: string;
	repeats: boolean;
	note?: string;
	due: boolean;
	/** The due day, YYYY-MM-DD. */
	due_on?: string;
	due_odometer?: number;
	/** Negative past the due day. */
	days_left?: number;
	/** Negative past the due odometer value. */
	distance_left?: number;
}
