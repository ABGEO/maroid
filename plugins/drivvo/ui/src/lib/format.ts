import type { Kind, Money, Units, VolumeUnit } from '../api';

const KIND_LABELS: Record<Kind, string> = {
	refuel: 'Refuel',
	maintenance: 'Maintenance',
	expense: 'Expense',
	income: 'Income',
	odometer: 'Odometer'
};

export const KINDS: Kind[] = ['refuel', 'maintenance', 'expense', 'income', 'odometer'];

export function kindLabel(kind: Kind): string {
	return KIND_LABELS[kind];
}

const VOLUME_LABELS: Record<VolumeUnit, string> = {
	l: 'L',
	gal_us: 'US gal',
	gal_imp: 'imp gal'
};

export function volumeLabel(units: Units): string {
	return VOLUME_LABELS[units.volume] ?? units.volume;
}

export function consumptionLabel(units: Units): string {
	const volume = volumeLabel(units);

	return units.consumption_shape === 'volume_per_100'
		? `${volume}/100 ${units.distance}`
		: `${units.distance}/${volume}`;
}

const amountFormat = new Intl.NumberFormat('en-US', {
	minimumFractionDigits: 2,
	maximumFractionDigits: 3
});
const wholeFormat = new Intl.NumberFormat('en-US', { maximumFractionDigits: 0 });
const tenthFormat = new Intl.NumberFormat('en-US', {
	minimumFractionDigits: 1,
	maximumFractionDigits: 1
});

/** An amount with its currency, as 1,500.00 GEL. */
export function money(value: Money): string {
	return `${amountFormat.format(value.amount)} ${value.currency}`;
}

/** An amount with no currency, for a column that names the currency in its head. */
export function amount(value: Money): string {
	return amountFormat.format(value.amount);
}

/** A distance or an odometer value, as 152,300 km. */
export function distance(value: number, units: Units): string {
	return `${wholeFormat.format(value)} ${units.distance}`;
}

/** A volume with two decimals, as 40.00 L. */
export function volume(value: number, units: Units): string {
	return `${value.toFixed(2)} ${volumeLabel(units)}`;
}

/** A consumption with one decimal, as 6.3 L/100 km. */
export function consumption(value: number, units: Units): string {
	return `${tenthFormat.format(value)} ${consumptionLabel(units)}`;
}

/** An amount per distance, as 0.30 GEL/km. */
export function moneyPerDistance(value: Money, units: Units): string {
	return `${money(value)}/${units.distance}`;
}

const dayFormat = new Intl.DateTimeFormat(undefined, { dateStyle: 'medium', timeZone: 'UTC' });

/** A day YYYY-MM-DD in the form of the browser locale. */
export function day(value: string): string {
	return dayFormat.format(new Date(`${value}T00:00:00Z`));
}

function plural(count: number, one: string, many: string): string {
	return count === 1 ? one : many;
}

/** What is left of a reminder: "in 12 days", "today", or "3 days past". */
export function daysLeft(days: number): string {
	if (days === 0) {
		return 'today';
	}

	const count = Math.abs(days);
	const unit = plural(count, 'day', 'days');

	return days > 0 ? `in ${count} ${unit}` : `${count} ${unit} past`;
}

/** What is left of a reminder: "in 1,200 km" or "250 km past". */
export function distanceLeft(left: number, units: Units): string {
	const shown = distance(Math.abs(left), units);

	return left >= 0 ? `in ${shown}` : `${shown} past`;
}
