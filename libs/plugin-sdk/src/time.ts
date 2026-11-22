// Each helper takes the time zone of the person. An undefined zone is the zone of the
// browser, because that is the default of Intl.

const DAY_MS = 86_400_000;

const wallFormats = new Map<string, Intl.DateTimeFormat>();

function wallFormat(timeZone: string | undefined): Intl.DateTimeFormat {
	const key = timeZone ?? '';
	let format = wallFormats.get(key);

	if (format === undefined) {
		format = new Intl.DateTimeFormat('en-US', {
			timeZone,
			hourCycle: 'h23',
			year: 'numeric',
			month: '2-digit',
			day: '2-digit',
			hour: '2-digit',
			minute: '2-digit',
			second: '2-digit'
		});
		wallFormats.set(key, format);
	}

	return format;
}

/** The wall time of a moment in the zone, as YYYY-MM-DDTHH:mm:ss. */
function wallTime(moment: number, timeZone: string | undefined): string {
	const part: Record<string, string> = {};

	for (const { type, value } of wallFormat(timeZone).formatToParts(moment)) {
		part[type] = value;
	}

	return `${part.year}-${part.month}-${part.day}T${part.hour}:${part.minute}:${part.second}`;
}

/** The offset of the zone at a moment, in milliseconds east of UTC. */
function offsetAt(moment: number, timeZone: string | undefined): number {
	const second = Math.floor(moment / 1000) * 1000;

	return Date.parse(`${wallTime(second, timeZone)}Z`) - second;
}

/** A moment of the hub, in the zone and the form of the browser locale. */
export function formatMoment(
	value: string | Date,
	timeZone?: string,
	options: Intl.DateTimeFormatOptions = {}
): string {
	return new Date(value).toLocaleString(undefined, { ...options, timeZone });
}

/**
 * The value of a datetime-local input, read as a wall time in the zone, as a UTC
 * moment. It answers undefined for an empty value. A repeated wall time takes the
 * first of its two moments. A wall time in a gap moves forward by the gap.
 */
export function toMoment(local: string, timeZone?: string): string | undefined {
	if (local === '') {
		return undefined;
	}

	const wall = local.length === 16 ? `${local}:00` : local.slice(0, 19);
	const asUTC = Date.parse(`${wall}Z`);
	const before = offsetAt(asUTC - DAY_MS, timeZone);
	const after = offsetAt(asUTC + DAY_MS, timeZone);
	const matches = [asUTC - before, asUTC - after].filter(
		(candidate) => wallTime(candidate, timeZone) === wall
	);
	const moment = matches.length > 0 ? Math.min(...matches) : asUTC - before;

	return new Date(moment).toISOString();
}

/** A moment as the value of a datetime-local input, YYYY-MM-DDTHH:mm, in the zone. */
export function toLocalInput(value: string | Date, timeZone?: string): string {
	return wallTime(new Date(value).getTime(), timeZone).slice(0, 16);
}

/** Today as YYYY-MM-DD, in the zone. */
export function today(timeZone?: string): string {
	return wallTime(Date.now(), timeZone).slice(0, 10);
}
