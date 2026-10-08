import { parse, stringify } from 'yaml';

import { ApiError, PROBLEM_TYPE, type Provider, type ProviderPreset } from '$lib/api';

/** The shortest local password, which the hub also checks. */
export const MIN_PASSWORD_LENGTH = 12;
/** The longest local password: bcrypt reads no further. */
export const MAX_PASSWORD_BYTES = 72;

const PRESET_LABELS: Record<ProviderPreset, string> = {
	local: 'Email',
	telegram: 'Telegram',
	oidc: 'OpenID Connect'
};

export function presetLabel(provider: Provider): string {
	return provider.preset ? PRESET_LABELS[provider.preset] : 'Dex configuration file';
}

export interface ParsedOptions {
	options?: Record<string, unknown>;
	error?: string;
}

/** Reads the YAML of the options. Empty text holds no option. */
export function parseOptions(text: string): ParsedOptions {
	if (text.trim() === '') {
		return { options: {} };
	}

	let value: unknown;

	try {
		value = parse(text);
	} catch (error) {
		return { error: error instanceof Error ? error.message : 'The text is not YAML.' };
	}

	if (value === null || value === undefined) {
		return { options: {} };
	}

	if (typeof value !== 'object' || Array.isArray(value)) {
		return { error: 'The options are a mapping of a key to a value.' };
	}

	return { options: value as Record<string, unknown> };
}

export function optionsText(options: Record<string, unknown> | undefined): string {
	return options && Object.keys(options).length > 0 ? stringify(options) : '';
}

/**
 * Answers the merge patch that turns the stored options into the edited ones. A key
 * that the edit drops goes out as null, which removes it.
 */
export function optionsPatch(
	stored: Record<string, unknown> | undefined,
	edited: Record<string, unknown>
): Record<string, unknown> | undefined {
	const patch: Record<string, unknown> = {};

	for (const key of Object.keys(stored ?? {})) {
		if (!(key in edited)) {
			patch[key] = null;
		}
	}

	for (const [key, value] of Object.entries(edited)) {
		if (JSON.stringify(stored?.[key]) !== JSON.stringify(value)) {
			patch[key] = value;
		}
	}

	return Object.keys(patch).length > 0 ? patch : undefined;
}

/** Splits scopes that a space or a comma separates. */
export function parseScopes(text: string): string[] {
	return text
		.split(/[\s,]+/)
		.map((scope) => scope.trim())
		.filter(Boolean);
}

/** Answers why the password is refused, or null for one that the hub accepts. */
export function passwordProblem(password: string, repeated: string): string | null {
	if ([...password].length < MIN_PASSWORD_LENGTH) {
		return `The password holds at least ${MIN_PASSWORD_LENGTH} characters.`;
	}

	if (new TextEncoder().encode(password).length > MAX_PASSWORD_BYTES) {
		return `The password holds at most ${MAX_PASSWORD_BYTES} bytes. A letter outside English can take more than one.`;
	}

	if (password !== repeated) {
		return 'The two passwords differ.';
	}

	return null;
}

/** Answers the detail of each failed field, keyed by its JSON Pointer. */
export function fieldErrors(error: unknown): Record<string, string> {
	if (!(error instanceof ApiError)) {
		return {};
	}

	return Object.fromEntries(error.fields.map((field) => [field.pointer, field.detail]));
}

/** Answers the sentence that a failed request shows to the administrator. */
export function failureMessage(error: unknown, fallback: string): string {
	if (!(error instanceof ApiError) || error.problem === null) {
		return fallback;
	}

	switch (error.problem.type) {
		case PROBLEM_TYPE.providerExists:
			return 'Dex already holds a provider with this identifier.';
		case PROBLEM_TYPE.providerStatic:
			return 'The configuration file of Dex holds this provider, so Maroid changes nothing on it.';
		case PROBLEM_TYPE.localAccountExists:
			return 'Another local account holds this email address, or the user already holds one.';
		case PROBLEM_TYPE.localProviderAbsent:
			return 'Add the Email provider first, on the page of the sign-in providers.';
		case PROBLEM_TYPE.identityLast:
			return 'This is the last way that the user signs in. Give them another account first.';
		case PROBLEM_TYPE.preconditionFailed:
			return 'Someone changed this provider after you opened it. Reload the page.';
		case PROBLEM_TYPE.notReady:
			return 'Dex does not answer. Try again in a moment.';
		default:
			return error.problem.errors?.[0]?.detail ?? error.problem.title;
	}
}
