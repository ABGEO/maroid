import type { SubscriberKind } from '../api';

const KIND_LABELS: Record<SubscriberKind, string> = {
	mobile: 'Mobile',
	home_line: 'Home line',
	other: 'Other'
};

export function kindLabel(kind: SubscriberKind): string {
	return KIND_LABELS[kind];
}

/** A count of seconds as mm:ss. A count of an hour or more keeps the minutes above 59. */
export function minutesAndSeconds(seconds: number): string {
	const whole = Math.max(0, Math.round(seconds));
	const minutes = Math.floor(whole / 60);

	return `${minutes}:${String(whole % 60).padStart(2, '0')}`;
}

export function lari(amount: number): string {
	return `${amount.toFixed(2)} ₾`;
}

/** An amount of data in MB with at most one decimal and no trailing zero. */
export function megabytes(value: number): string {
	return String(Math.round(value * 10) / 10);
}

export function renewal(daysLeft: number): string {
	if (daysLeft <= 0) {
		return 'renews today';
	}

	return daysLeft === 1 ? 'renews in 1 day' : `renews in ${daysLeft} days`;
}
