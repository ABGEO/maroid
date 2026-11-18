/** A moment of the hub, in the zone and the form of the browser. */
export function formatMoment(value: string): string {
	return new Date(value).toLocaleString();
}

/** The value of a datetime-local input as a UTC moment, or undefined when empty. */
export function toMoment(local: string): string | undefined {
	return local === '' ? undefined : new Date(local).toISOString();
}

/** A moment as the value of a datetime-local input, in the zone of the browser. */
export function toLocalInput(value: string | Date): string {
	const moment = new Date(value);
	moment.setMinutes(moment.getMinutes() - moment.getTimezoneOffset());

	return moment.toISOString().slice(0, 16);
}

/** The current time as the value of a datetime-local input, the latest it takes. */
export function localNow(): string {
	return toLocalInput(new Date());
}

/** A day as YYYY-MM-DD, in the long form of the browser. */
export function formatDay(day: string): string {
	return new Date(`${day}T00:00`).toLocaleDateString(undefined, { dateStyle: 'long' });
}

/** How long ago a YYYY-MM-DD day was: "today", "3 weeks ago", "1 year ago". */
export function ageOf(day: string): string {
	const days = Math.round((Date.now() - new Date(`${day}T00:00`).getTime()) / 86_400_000);
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

/** Today as YYYY-MM-DD, in the zone of the browser. */
export function today(): string {
	return toLocalInput(new Date()).slice(0, 10);
}

/** A count of days, with the singular for one. */
export function formatDays(count: number): string {
	return count === 1 ? '1 day' : `${count} days`;
}

/** The days until a care interval becomes due: "tomorrow" for one, else "in N days". */
export function dueLabel(dueIn: number): string {
	return dueIn === 1 ? 'tomorrow' : `in ${dueIn} days`;
}
