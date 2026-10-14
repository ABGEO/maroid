import type { Transaction } from '../api';

/** The usage and the charge of one reading. */
export interface ChartPoint {
	/** The date of the reading, YYYY-MM-DD. */
	date: string;
	/** The usage, in kWh. */
	usage: number;
	/** The charge, in lari. */
	amount: number;
}

/**
 * The readings among the transactions, the oldest first. Only a charge carries a usage.
 * Telmico splits a reading in two charges on one date when the tariff changes, so the
 * charges of one date add up to one reading.
 */
export function chartPoints(transactions: Transaction[]): ChartPoint[] {
	const byDate = new Map<string, ChartPoint>();

	for (const transaction of transactions) {
		if (transaction.usage === undefined) {
			continue;
		}

		const point = byDate.get(transaction.transaction_date) ?? {
			date: transaction.transaction_date,
			usage: 0,
			amount: 0
		};

		point.usage += transaction.usage;
		point.amount += transaction.amount?.amount ?? 0;
		byDate.set(point.date, point);
	}

	return [...byDate.values()].sort((a, b) => a.date.localeCompare(b.date));
}

/** An amount of energy with at most one decimal and no trailing zero. */
export function kilowattHours(value: number): string {
	return String(Math.round(value * 10) / 10);
}
