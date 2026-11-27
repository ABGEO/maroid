/** The periods that a member chooses for a chart. */
export type Period = 'year' | 'three_years' | 'all';

export const DEFAULT_PERIOD: Period = 'year';

export const PERIODS: ReadonlyArray<{ value: Period; label: string }> = [
	{ value: 'year', label: '12 months' },
	{ value: 'three_years', label: '3 years' },
	{ value: 'all', label: 'All' }
];

/** The months before the current month that each period reaches back. */
const MONTHS_BACK: Record<Exclude<Period, 'all'>, number> = { year: 11, three_years: 35 };

/** The first day of the month of a date YYYY-MM-DD, moved by a count of months. */
export function monthStart(date: string, shift = 0): string {
	const year = Number(date.slice(0, 4));
	const month = Number(date.slice(5, 7)) - 1 + shift;
	const first = new Date(Date.UTC(year, month, 1));

	return first.toISOString().slice(0, 10);
}

/** The first date of a period that ends in the month of today, or undefined for every month. */
export function periodFrom(period: Period, today: string): string | undefined {
	return period === 'all' ? undefined : monthStart(today, -MONTHS_BACK[period]);
}

/** The first day of each month from the month of from to the month of to, both included. */
export function monthSlots(from: string, to: string): string[] {
	const last = monthStart(to);
	const slots: string[] = [];

	for (let slot = monthStart(from); slot <= last; slot = monthStart(slot, 1)) {
		slots.push(slot);
	}

	return slots;
}
