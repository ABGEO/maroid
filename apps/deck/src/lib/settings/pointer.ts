/**
 * pathOf turns a JSON Pointer into the path of segments that a form reads.
 * The pointer arrives in the fragment form of RFC 6901, so it starts with `#`,
 * and RFC 6901 escapes a tilde as `~0` and a slash as `~1`.
 */
export function pathOf(pointer: string): string[] {
	const body = pointer.startsWith('#') ? pointer.slice(1) : pointer;

	if (body === '' || body === '/') {
		return [];
	}

	return body
		.replace(/^\//, '')
		.split('/')
		.map((segment) => segment.replaceAll('~1', '/').replaceAll('~0', '~'));
}
