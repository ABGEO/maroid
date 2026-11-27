import type { ContributionFilter } from '../api';

const KEYS = ['employer', 'from', 'to'] as const;

/** The filter that the query of the address names. An empty value names no filter. */
export function filterOfAddress(search: string): ContributionFilter {
	const query = new URLSearchParams(search);
	const filter: ContributionFilter = {};

	for (const key of KEYS) {
		const value = query.get(key);
		if (value) {
			filter[key] = value;
		}
	}

	return filter;
}

/**
 * Writes the filter to the query of the address with no new history entry. The state of
 * the deck router stays, so the back button and the navigation of the deck still work.
 */
export function keepInAddress(filter: ContributionFilter): void {
	const url = new URL(window.location.href);

	for (const key of KEYS) {
		const value = filter[key];
		if (value) {
			url.searchParams.set(key, value);
		} else {
			url.searchParams.delete(key);
		}
	}

	history.replaceState(history.state, '', url);
}
