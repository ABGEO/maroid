import type { Crumb } from '$lib/navigation/breadcrumbs';

const BRAND = 'Maroid';

/** The browser title for a page name. A page without a name reads as the brand alone. */
export function documentTitle(name?: string): string {
	return name ? `${name} | ${BRAND}` : BRAND;
}

/**
 * The browser title for a trail: the name of its last crumb. The hub alone and a
 * page that has not named itself yet read as the brand alone.
 */
export function trailTitle(crumbs: Crumb[]): string {
	const last = crumbs.at(-1);
	if (crumbs.length < 2 || !last || last.pending) {
		return documentTitle();
	}

	return documentTitle(last.label);
}
