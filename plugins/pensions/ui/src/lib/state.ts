import type { SummaryView } from '../api';

/** What a page shows: the data, the note that no collection ran yet, or the setup. */
export type PageState = 'ready' | 'empty' | 'not_set_up';

/** A stored summary decides first, so a member who disconnected keeps the collected data. */
export function pageState(view: SummaryView): PageState {
	if (view.summary !== undefined) {
		return 'ready';
	}

	return view.connected ? 'empty' : 'not_set_up';
}
