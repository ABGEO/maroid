export type ConnectionState = 'not_connected' | 'connected';

/** An amount of money, the money object of Zalando rule 173. */
export interface Money {
	amount: number;
	/** The ISO 4217 code. The pension service reports in lari alone. */
	currency: string;
}

export interface Holder {
	/** The first name and the last name, with one space. */
	full_name: string;
	asset_value: Money;
}

export interface Connection {
	state: ConnectionState;
	credentials_stored: boolean;
	/** Present when the user is connected. */
	holder?: Holder;
}

/** The sum of the contributions of each source, and of every source. */
export interface SourceSums {
	person: Money;
	employer: Money;
	state: Money;
	total: Money;
}

/** The account summary that the last collection stored. */
export interface Summary {
	asset_value: Money;
	units: number;
	gain: Money;
	withheld_amount: Money;
	contributions: SourceSums;
	/** The moment of the collection, RFC 3339. */
	collected_at: string;
}

/** What every page reads first. */
export interface SummaryView {
	/** Maroid holds a session of the user. */
	connected: boolean;
	/** Absent until a collection stores one. */
	summary?: Summary;
}

export interface AssetValue {
	/** YYYY-MM-DD. */
	valued_on: string;
	asset_value: Money;
	units: number;
	unit_price: number;
}

export interface ContributionMonth {
	/** YYYY-MM. */
	month: string;
	amount: Money;
}

export interface Employer {
	code: string;
	name: string;
}

/** One contribution. `source` and `type` are as the pension service gives them. */
export interface Contribution {
	id: string;
	/** YYYY-MM-DD. */
	paid_on: string;
	employer?: Employer;
	source: string;
	type: string;
	/** YYYY-MM. */
	salary_month?: string;
	gross_salary: Money;
	amount?: Money;
}

export interface ContributionTotal {
	amount: Money;
	count: number;
}

/** The narrowing of the contributions. An absent member keeps every contribution. */
export interface ContributionFilter {
	employer?: string;
	/** YYYY-MM-DD, included. */
	from?: string;
	/** YYYY-MM-DD, included. */
	to?: string;
}

export interface PortfolioShare {
	portfolio_id: number;
	portfolio_name: string;
	asset_manager_name: string;
	ticker: string;
	risk_tier_name: string;
	/** The chosen share, a percent from 0 to 100. */
	target_share: number;
	/** The current share, a percent from 0 to 100. */
	current_share: number;
	amount: Money;
	units: number;
	unit_price: number;
}

export interface UnitPrice {
	portfolio_id: number;
	/** YYYY-MM-DD, the last priced day of its week. */
	priced_on: string;
	unit_price: number;
}
