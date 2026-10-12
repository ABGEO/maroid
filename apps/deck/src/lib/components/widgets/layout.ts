import { resolve } from '$app/paths';

import type { WidgetSize } from '$lib/api/types';

/**
 * The space of each size on a grid of one, two, or four columns. A remote ships its
 * own utilities, and they stay in the page after its first page opens, so a later
 * `sm:` rule of a remote beats an `lg:` rule of the deck. The important flag keeps
 * the order of the deck.
 */
const SPANS: Record<WidgetSize, string> = {
	xs: 'col-span-1! row-span-1!',
	s: 'sm:col-span-2! row-span-1!',
	m: 'sm:col-span-2! row-span-2!',
	l: 'sm:col-span-2! lg:col-span-4! row-span-2!'
};

/** The columns of the grid, with the important flag for the reason of SPANS. */
export const GRID_COLUMNS = 'grid-cols-1! sm:grid-cols-2! lg:grid-cols-4!';

export function spanOf(size: WidgetSize): string {
	return SPANS[size];
}

/** How many items a list shows: 2 in one row, 5 in two. */
export function itemsOf(size: WidgetSize): number {
	return size === 'xs' || size === 's' ? 2 : 5;
}

/** The deck address of a path of the plugin, in the workspace. */
export function linkOf(workspaceId: string, pluginId: string, path: string): string {
	return resolve('/(dashboard)/w/[workspace]/plugins/[plugin]/[...path]', {
		workspace: workspaceId,
		plugin: pluginId,
		path: path.replace(/^\//, '')
	});
}
