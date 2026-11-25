import { today } from '@maroid/plugin-sdk';

import type { Period } from '../api';

export type PeriodChoice = 'month' | 'last12' | 'year' | 'all' | 'range';

export const PERIOD_CHOICES: { value: PeriodChoice; label: string }[] = [
	{ value: 'month', label: 'This month' },
	{ value: 'last12', label: 'Last 12 months' },
	{ value: 'year', label: 'This year' },
	{ value: 'all', label: 'All time' },
	{ value: 'range', label: 'Range' }
];

/** The first day of the month that lies a count of months before the month of the day. */
function monthStart(day: string, monthsBack: number): string {
	const [year, month] = day.split('-').map(Number);

	return new Date(Date.UTC(year, month - 1 - monthsBack, 1)).toISOString().slice(0, 10);
}

/**
 * The days of a choice in the zone of the member. A range takes the two days that the
 * member picked, and an empty day leaves that end open.
 */
export function periodOf(
	choice: PeriodChoice,
	timeZone: string | undefined,
	range: Period = {}
): Period {
	const now = today(timeZone);

	switch (choice) {
		case 'month':
			return { from: monthStart(now, 0), to: now };
		case 'last12':
			return { from: monthStart(now, 11), to: now };
		case 'year':
			return { from: `${now.slice(0, 4)}-01-01`, to: now };
		case 'range':
			return { from: range.from || undefined, to: range.to || undefined };
		case 'all':
			return {};
	}
}
