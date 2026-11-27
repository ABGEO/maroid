import type { ChargeMonth } from '../api';
import { monthSlots } from './period';

/** The columns and the line of the overview chart, each keyed by a month slot YYYY-MM-01. */
export interface ChargeSeries {
	slots: string[];
	columns: Map<string, number>;
	line: Map<string, number>;
}

/**
 * One slot for each month of the period. A column is the sum of the charges of its month,
 * and a point is the balance after its latest transaction. A month with no transaction
 * keeps its slot with no column and no point. With no from, the period starts at the
 * earliest month of the data.
 */
export function chargeSeries(
	months: ChargeMonth[],
	from: string | undefined,
	today: string
): ChargeSeries {
	const earliest = months.map((month) => `${month.month}-01`).sort()[0];
	const slots = monthSlots(from ?? earliest ?? today, today);
	const inPeriod = new Set(slots);

	const columns = new Map<string, number>();
	const line = new Map<string, number>();

	for (const month of months) {
		const slot = `${month.month}-01`;
		if (!inPeriod.has(slot)) {
			continue;
		}

		if (month.charges.amount > 0) {
			columns.set(slot, month.charges.amount);
		}

		line.set(slot, month.balance.amount);
	}

	return { slots, columns, line };
}
