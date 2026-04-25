const STORAGE_KEY = 'maroid.theme';

export type ThemePreference = 'auto' | 'maroid' | 'maroid-dusk';

const VALID: readonly ThemePreference[] = ['auto', 'maroid', 'maroid-dusk'];

function isThemePreference(v: unknown): v is ThemePreference {
	return typeof v === 'string' && (VALID as readonly string[]).includes(v);
}

function read(): ThemePreference {
	if (typeof localStorage === 'undefined') {
		return 'auto';
	}

	const storedValue = localStorage.getItem(STORAGE_KEY);
	return isThemePreference(storedValue) ? storedValue : 'auto';
}

function apply(pref: ThemePreference): void {
	if (typeof document === 'undefined') {
		return;
	}

	if (pref === 'auto') {
		document.documentElement.removeAttribute('data-theme');
	} else {
		document.documentElement.setAttribute('data-theme', pref);
	}
}

export const themeState = $state<{ pref: ThemePreference }>({ pref: read() });

export function setTheme(next: ThemePreference): void {
	themeState.pref = next;
	apply(next);

	try {
		localStorage.setItem(STORAGE_KEY, next);
	} catch {
		// ignore — storage may be unavailable (private mode, etc.)
	}
}
