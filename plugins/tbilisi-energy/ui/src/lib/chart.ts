import type { Transaction } from '../api';

/** The usage and the charge of one reading. */
export interface ChartPoint {
	/** The date of the reading, YYYY-MM-DD. */
	date: string;
	/** The usage, in cubic metres. */
	usage: number;
	/** The charge, in lari. */
	amount: number;
}

/** The readings among the transactions, the oldest first. A payment carries no usage. */
export function chartPoints(transactions: Transaction[]): ChartPoint[] {
	return transactions
		.flatMap((transaction) =>
			transaction.usage === undefined
				? []
				: [
						{
							date: transaction.transaction_date,
							usage: transaction.usage,
							amount: transaction.amount.amount
						}
					]
		)
		.sort((a, b) => a.date.localeCompare(b.date));
}
