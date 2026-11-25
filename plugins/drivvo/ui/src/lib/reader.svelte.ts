export interface Reader<T> {
	readonly value: T | null;
	readonly status: 'loading' | 'ready' | 'error';
	/** The error of the last read that failed. */
	readonly error: unknown;
	/** Runs the read. A read that a later one superseded changes nothing. */
	load(read: () => Promise<T | null>): Promise<void>;
}

/** Holds the answer of the latest read of one resource. */
export function createReader<T>(): Reader<T> {
	let value = $state<T | null>(null);
	let status = $state<'loading' | 'ready' | 'error'>('loading');
	let error = $state<unknown>(null);
	let latest = 0;

	async function load(read: () => Promise<T | null>): Promise<void> {
		const request = ++latest;
		status = 'loading';

		try {
			const answered = await read();
			// A null answer means the client already sent the person to sign in.
			if (request === latest && answered !== null) {
				value = answered;
				status = 'ready';
			}
		} catch (failure) {
			if (request === latest) {
				console.error('Failed to load', failure);
				error = failure;
				status = 'error';
			}
		}
	}

	return {
		get value() {
			return value;
		},
		get status() {
			return status;
		},
		get error() {
			return error;
		},
		load
	};
}
