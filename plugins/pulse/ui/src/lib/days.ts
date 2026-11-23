const DAY_MS = 86_400_000;

/** The day count days after a day, both as YYYY-MM-DD. */
export function addDays(day: string, count: number): string {
	return new Date(Date.parse(`${day}T00:00:00Z`) + count * DAY_MS).toISOString().slice(0, 10);
}

/** A day as YYYY-MM-DD, in the medium form of the browser. */
export function formatDay(day: string): string {
	return new Date(`${day}T00:00:00Z`).toLocaleDateString(undefined, {
		dateStyle: 'medium',
		timeZone: 'UTC'
	});
}
