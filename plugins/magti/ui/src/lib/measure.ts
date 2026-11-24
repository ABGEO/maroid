import type { TrafficDay } from '../api';

/** One measure of a day, drawn as one series of columns. */
export interface Measure {
	key: string;
	name: string;
	read: (day: TrafficDay) => number | undefined;
	color: 'primary' | 'secondary';
}
