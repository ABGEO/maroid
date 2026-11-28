export const RETURN_ADDRESS_PARAMETER = 'next';

const HOME = '/';

/** The routes that start or end a sign in. A return address never names one. */
const SIGN_IN_PATHS = new Set(['/auth/callback', '/signed-out', '/invite']);

function pathQueryFragment(address: URL): string {
	return address.pathname + address.search + address.hash;
}

/**
 * The router decodes a path and drops a trailing slash before it picks a route,
 * so `/signed-out/` and `/signed%2Dout` both open the page that follows a sign
 * out. A path that does not decode counts as a route of the sign in, so the
 * check fails closed.
 */
function isSignInPath(pathname: string): boolean {
	let routed: string;
	try {
		routed = pathname.split('%25').map(decodeURI).join('%25');
	} catch {
		return true;
	}

	return SIGN_IN_PATHS.has(routed.replace(/\/+$/, '') || HOME);
}

/** The return address of the open address, or undefined when it is a route of the sign in. */
export function returnAddressOf(open: URL): string | undefined {
	if (isSignInPath(open.pathname)) {
		return undefined;
	}

	return pathQueryFragment(open);
}

/**
 * The address that the landing opens after a sign in. The raw value comes from
 * the address bar, so a link that anyone writes reaches this check. The parser
 * of the browser reads `//host`, `/\host`, and `javascript:` the way a browser
 * opens them, and the comparison of origins refuses each one. The result keeps
 * its origin, because `/.//host` resolves to the path `//host`, and a browser
 * opens that path alone as a foreign host.
 */
export function addressAfterSignIn(raw: string | null, origin: string): string {
	if (!raw) {
		return HOME;
	}

	let address: URL;
	try {
		address = new URL(raw, origin);
	} catch {
		return HOME;
	}

	if (address.origin !== origin || isSignInPath(address.pathname)) {
		return HOME;
	}

	return address.href;
}
