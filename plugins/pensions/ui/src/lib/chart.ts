import type { AssetValue, ContributionMonth } from '../api';
import { monthSlots, monthStart } from './period';

/** The columns and the line of the overview chart, each keyed by a month slot YYYY-MM-01. */
export interface ValueSeries {
	slots: string[];
	columns: Map<string, number>;
	line: Map<string, number>;
}

/**
 * One slot for each month of the period. A column is the sum of the contributions of its
 * month. A point is the last asset value within its month. With no from, the period starts
 * at the earliest month of the data.
 */
export function valueSeries(
	values: AssetValue[],
	months: ContributionMonth[],
	from: string | undefined,
	today: string
): ValueSeries {
	const earliest = [
		...values.map((value) => value.valued_on),
		...months.map((m) => `${m.month}-01`)
	]
		.sort()
		.at(0);
	const slots = monthSlots(from ?? earliest ?? today, today);
	const inPeriod = new Set(slots);

	const columns = new Map<string, number>();
	for (const month of months) {
		const slot = `${month.month}-01`;
		if (inPeriod.has(slot)) {
			columns.set(slot, month.amount.amount);
		}
	}

	const line = new Map<string, number>();
	for (const value of [...values].sort((a, b) => a.valued_on.localeCompare(b.valued_on))) {
		const slot = monthStart(value.valued_on);
		if (inPeriod.has(slot)) {
			line.set(slot, value.asset_value.amount);
		}
	}

	return { slots, columns, line };
}
