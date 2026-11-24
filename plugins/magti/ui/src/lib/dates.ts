/** A date YYYY-MM-DD moved by a count of days, in the calendar alone. */
export function addDays(date: string, days: number): string {
	const moved = new Date(`${date}T00:00:00Z`);
	moved.setUTCDate(moved.getUTCDate() + days);

	return moved.toISOString().slice(0, 10);
}

/** A month YYYY-MM moved by a count of months. */
export function addMonths(month: string, months: number): string {
	const [year, number] = month.split('-').map(Number);
	const moved = new Date(Date.UTC(year, number - 1 + months, 1));

	return moved.toISOString().slice(0, 7);
}

const TBILISI = 'Asia/Tbilisi';

/** The current month of Tbilisi, YYYY-MM, the month that the routes of money count in. */
export function currentTbilisiMonth(): string {
	return new Intl.DateTimeFormat('en-CA', { timeZone: TBILISI, year: 'numeric', month: '2-digit' })
		.format(Date.now())
		.slice(0, 7);
}

/** Today in Tbilisi, YYYY-MM-DD, the day that the traffic of the portal counts in. */
export function tbilisiToday(): string {
	return new Intl.DateTimeFormat('en-CA', {
		timeZone: TBILISI,
		year: 'numeric',
		month: '2-digit',
		day: '2-digit'
	}).format(Date.now());
}
