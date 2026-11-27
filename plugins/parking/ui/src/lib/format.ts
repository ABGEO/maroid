import { formatMoment } from '@maroid/plugin-sdk';

import type { Money, TransactionKind } from '../api';

const SECONDS_IN_MINUTE = 60;
const MINUTES_IN_HOUR = 60;

const money = new Intl.NumberFormat(undefined, {
	minimumFractionDigits: 2,
	maximumFractionDigits: 2
});

/** Writes an amount with two decimals and the lari sign. */
export function formatMoney(value: Money): string {
	const sign = value.currency === 'GEL' ? '₾' : value.currency;

	return `${money.format(value.amount)} ${sign}`;
}

/** Writes a duration as hours and minutes, such as 1 h 05 min. */
export function formatDuration(seconds: number): string {
	const minutes = Math.round(seconds / SECONDS_IN_MINUTE);
	const hours = Math.floor(minutes / MINUTES_IN_HOUR);
	const rest = minutes % MINUTES_IN_HOUR;

	if (hours === 0) {
		return `${rest} min`;
	}

	return `${hours} h ${String(rest).padStart(2, '0')} min`;
}

/** The date of a moment, in the zone. */
export function formatDay(moment: string, timeZone: string | undefined): string {
	return formatMoment(moment, timeZone, { dateStyle: 'medium' });
}

/** The time of a moment, in the zone. */
export function formatTime(moment: string, timeZone: string | undefined): string {
	return formatMoment(moment, timeZone, { timeStyle: 'short' });
}

export const KIND_LABELS: Record<TransactionKind, string> = {
	top_up: 'Top up',
	parking_payment: 'Parking payment',
	permit_purchase: 'Permit purchase',
	other: 'Other'
};

/** The sign of a kind: a top up adds to the balance, a charge takes from it, another kind has none. */
export function signOf(kind: TransactionKind): '+' | '−' | '' {
	switch (kind) {
		case 'top_up':
			return '+';
		case 'parking_payment':
		case 'permit_purchase':
			return '−';
		default:
			return '';
	}
}
