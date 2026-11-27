import type { Page } from '@maroid/plugin-sdk';

/** The page size of a read of every page, the largest that a collection serves. */
export const PAGE_LIMIT = 100;

/** Every item of a collection, read page by page from the first, or null when the first page is. */
export async function readAll<T>(
	read: (link?: string) => Promise<Page<T> | null>
): Promise<T[] | null> {
	const items: T[] = [];
	let link: string | undefined;

	do {
		const page = await read(link);
		if (page === null) {
			return link === undefined ? null : items;
		}

		items.push(...page.items);
		link = page.next;
	} while (link !== undefined);

	return items;
}
