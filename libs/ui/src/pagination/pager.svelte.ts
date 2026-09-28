export interface PageLinks {
	next?: string;
	prev?: string;
}

export type PageStatus = 'loading' | 'ready' | 'error';

export type PageReader<P extends PageLinks> = (link?: string) => Promise<P | null>;

export interface Pager<P extends PageLinks> {
	readonly page: P | null;
	readonly status: PageStatus;
	readonly hasPrevious: boolean;
	readonly hasNext: boolean;

	/** Reads the page that the pager last asked for, the first page at the start. */
	reload(): Promise<void>;
	next(): Promise<void>;
	previous(): Promise<void>;
}

export function createPager<P extends PageLinks>(read: PageReader<P>): Pager<P> {
	let page = $state<P | null>(null);
	let status = $state<PageStatus>('loading');
	let requested: string | undefined;
	let latestRequest = 0;

	async function load(link?: string): Promise<void> {
		const request = ++latestRequest;
		const superseded = () => request !== latestRequest;

		requested = link;
		status = 'loading';

		try {
			const answered = await read(link);
			// A null answer means the client already sent the person to sign in.
			if (superseded() || answered === null) {
				return;
			}

			page = answered;
			status = 'ready';
		} catch (error) {
			if (superseded()) {
				return;
			}

			console.error('Failed to load a page', error);
			status = 'error';
		}
	}

	async function follow(link: string | undefined): Promise<void> {
		if (link !== undefined && status !== 'loading') {
			await load(link);
		}
	}

	return {
		get page() {
			return page;
		},
		get status() {
			return status;
		},
		get hasPrevious() {
			return page?.prev !== undefined;
		},
		get hasNext() {
			return page?.next !== undefined;
		},
		reload: () => load(requested),
		next: () => follow(page?.next),
		previous: () => follow(page?.prev)
	};
}
