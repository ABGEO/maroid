import type { RouteParams } from '@maroid/plugin-sdk';

/** The page route of a remote that an address matched, with the values of its parameters. */
export interface RouteMatch {
	key: string;
	params: RouteParams;
	parameterized: boolean;
}

const PARAMETER = /^:([A-Za-z][A-Za-z0-9_]*)$/;

function segmentsOf(path: string): string[] {
	const trimmed = path.replace(/^\/+|\/+$/g, '');

	return trimmed === '' ? [] : trimmed.split('/');
}

function decode(segment: string): string | null {
	try {
		return decodeURIComponent(segment);
	} catch {
		return null;
	}
}

function isFixed(segment: string | undefined): boolean {
	return segment !== undefined && !PARAMETER.test(segment);
}

/** The values that the key takes from the raw segments of an address, or null when it does not match. */
function bind(key: string, address: string[]): RouteParams | null {
	const pattern = segmentsOf(key);
	if (pattern.length !== address.length) {
		return null;
	}

	const params: RouteParams = {};

	for (const [index, part] of pattern.entries()) {
		const value = decode(address[index]);
		if (value === null || value === '') {
			return null;
		}

		const name = PARAMETER.exec(part)?.[1];
		if (name !== undefined) {
			params[name] = value;
		} else if (part !== value) {
			return null;
		}
	}

	return params;
}

/**
 * Orders two keys that match one address: at the first segment where one key is
 * fixed and the other takes a parameter, the fixed one comes first. Zero means the
 * two keys match the same addresses.
 */
function precedence(left: string, right: string): number {
	const rightSegments = segmentsOf(right);

	for (const [index, part] of segmentsOf(left).entries()) {
		const leftFixed = isFixed(part);
		if (leftFixed !== isFixed(rightSegments[index])) {
			return leftFixed ? -1 : 1;
		}
	}

	return 0;
}

/**
 * The page route that a raw address opens, or null when no key matches. Two keys
 * that match the same addresses are a defect of the remote, because the order of
 * the declaration would then pick the page.
 */
export function matchRoute(keys: string[], path: string): RouteMatch | null {
	const address = segmentsOf(path);
	const candidates = keys.flatMap((key) => {
		const params = bind(key, address);

		return params === null ? [] : [{ key, params }];
	});

	candidates.sort((left, right) => precedence(left.key, right.key));

	const [best, runnerUp] = candidates;
	if (best === undefined) {
		return null;
	}

	if (runnerUp !== undefined && precedence(best.key, runnerUp.key) === 0) {
		throw new Error(
			`The remote declares two routes for one address: "${best.key}" and "${runnerUp.key}"`
		);
	}

	return { key: best.key, params: best.params, parameterized: Object.keys(best.params).length > 0 };
}

/**
 * The navigation entry that the address sits under: the entry whose segments lead
 * the address for the most segments, or null when no entry leads it.
 */
export function activeEntry(paths: string[], path: string): string | null {
	const address = segmentsOf(path);
	let best: string | null = null;
	let bestLength = -1;

	for (const entry of paths) {
		const segments = segmentsOf(entry);
		const leads =
			segments.length <= address.length &&
			segments.every((segment, index) => segment === address[index]);

		if (leads && segments.length > bestLength) {
			best = entry;
			bestLength = segments.length;
		}
	}

	return best;
}
