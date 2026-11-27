import type { Money } from '../api';

const money = new Intl.NumberFormat(undefined, {
	minimumFractionDigits: 2,
	maximumFractionDigits: 2
});

const percent = new Intl.NumberFormat(undefined, { maximumFractionDigits: 2 });

const units = new Intl.NumberFormat(undefined, { maximumFractionDigits: 4 });

/** Writes an amount with two decimals and the lari sign. */
export function formatMoney(value: Money): string {
	const sign = value.currency === 'GEL' ? '₾' : value.currency;

	return `${money.format(value.amount)} ${sign}`;
}

/** Writes a percent from 0 to 100 with at most two decimals. */
export function formatPercent(value: number): string {
	return `${percent.format(value)}%`;
}

/** Writes a count of units or a unit price with at most four decimals. */
export function formatUnits(value: number): string {
	return units.format(value);
}

/** A date YYYY-MM-DD as dd/mm/yyyy, with no conversion of time zone. */
export function formatDate(date: string): string {
	const [year, month, day] = date.split('-');

	return `${day}/${month}/${year}`;
}

/** A month YYYY-MM as mm/yyyy. */
export function formatMonth(month: string): string {
	const [year, number] = month.split('-');

	return `${number}/${year}`;
}
