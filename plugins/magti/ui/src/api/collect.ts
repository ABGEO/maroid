import type { ApiClient, Page } from '@maroid/plugin-sdk';

/** The most rows that one page answers, so a year of days takes four pages. */
export const SERIES_PAGE_LIMIT = 100;

/** Reads every row across the pages. It answers null when the client sent the person to sign in. */
export async function readEvery<T>(
	api: ApiClient,
	path: string,
	params: Record<string, string | number>
): Promise<T[] | null> {
	const rows: T[] = [];

	let answered = await api.get<Page<T>>(path, { params });

	while (answered !== null) {
		rows.push(...answered.items);

		if (answered.next === undefined) {
			return rows;
		}

		answered = await api.follow<Page<T>>(answered.next);
	}

	return null;
}
