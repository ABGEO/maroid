import type { Page } from "@maroid/plugin-sdk";

/** Every item of a collection, read page by page from the first. */
export async function readAll<T>(
  read: (link?: string) => Promise<Page<T> | null>,
): Promise<T[]> {
  const items: T[] = [];
  let link: string | undefined;

  do {
    const page = await read(link);
    if (page === null) {
      return items;
    }

    items.push(...page.items);
    link = page.next;
  } while (link !== undefined);

  return items;
}
