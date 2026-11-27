/** The filter that the query of the address names, for the keys. An empty value names no filter. */
export function filterOfAddress<K extends string>(
	search: string,
	keys: readonly K[]
): Partial<Record<K, string>> {
	const query = new URLSearchParams(search);
	const filter: Partial<Record<K, string>> = {};

	for (const key of keys) {
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
export function keepInAddress<K extends string>(
	filter: Partial<Record<K, string>>,
	keys: readonly K[]
): void {
	const url = new URL(window.location.href);

	for (const key of keys) {
		const value = filter[key];
		if (value) {
			url.searchParams.set(key, value);
		} else {
			url.searchParams.delete(key);
		}
	}

	history.replaceState(history.state, '', url);
}
