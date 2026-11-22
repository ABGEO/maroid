import { toLocalInput, today } from '@maroid/plugin-sdk';

export { formatMoment, toLocalInput, toMoment, today } from '@maroid/plugin-sdk';

/** The current time as the value of a datetime-local input, in the zone, the latest it takes. */
export function localNow(timeZone?: string): string {
	return toLocalInput(new Date(), timeZone);
}

/** A day as YYYY-MM-DD, in the long form of the browser. */
export function formatDay(day: string): string {
	return new Date(`${day}T00:00`).toLocaleDateString(undefined, { dateStyle: 'long' });
}

/** How long ago a YYYY-MM-DD day was, from today in the zone: "today", "3 weeks ago". */
export function ageOf(day: string, timeZone?: string): string {
	const days = Math.round((Date.parse(today(timeZone)) - Date.parse(day)) / 86_400_000);
	const format = new Intl.RelativeTimeFormat(undefined, { numeric: 'auto' });

	if (days < 14) {
		return format.format(-days, 'day');
	}

	if (days < 60) {
		return format.format(-Math.round(days / 7), 'week');
	}

	if (days < 365) {
		return format.format(-Math.round(days / 30), 'month');
	}

	return format.format(-Math.round(days / 365), 'year');
}

/** A count of days, with the singular for one. */
export function formatDays(count: number): string {
	return count === 1 ? '1 day' : `${count} days`;
}

/** The days until a care interval becomes due: "tomorrow" for one, else "in N days". */
export function dueLabel(dueIn: number): string {
	return dueIn === 1 ? 'tomorrow' : `in ${dueIn} days`;
}
