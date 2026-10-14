export type ConnectionState = 'not_connected' | 'connected';

export interface Customer {
	/** The number of the customer at Telmico. */
	customer_number: string;
	/** The label that the holder gave the customer. */
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

/** An amount of money, the money object of Zalando rule 173. */
export interface Money {
	amount: number;
	/** The ISO 4217 code. Telmico bills in lari alone. */
	currency: string;
}

/** One transaction. A value that the kind of the transaction does not hold is absent. */
export interface Transaction {
	id: string;
	/** The date of the transaction, YYYY-MM-DD. */
	transaction_date: string;
	/** The name of the operation, as Telmico gives it. */
	operation: string;
	/** The amount. A payment is below zero. */
	amount?: Money;
	/** The balance after the transaction. */
	balance?: Money;
	/** The usage of a charge, in kWh. Only a charge carries it. */
	usage?: number;
	/** The value of the meter, in kWh. */
	meter_value?: number;
	/** The tariff of a charge, in lari for each kWh. */
	tariff?: number;
}

/** One bill of a month. */
export interface Bill {
	id: string;
	/** The month that the bill covers, YYYY-MM. */
	month: string;
	bill_number: string;
	/** The date of issue, YYYY-MM-DD. */
	issued_on: string;
	/** The address of the download of the document, at the hub. */
	document: string;
}
