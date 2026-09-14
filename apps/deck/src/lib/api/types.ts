export interface User {
	name: string;
	picture: string;
}

export interface UIRoute {
	path: string;
	label: string;
}

export interface UIManifest {
	name: string;
	routes: UIRoute[];
}

export interface Plugin {
	id: string;
	version: string;
	settings: boolean;
	ui?: UIManifest;
}

/** One property of a settings schema. The hub infers it from the struct of the plugin. */
export interface SchemaProperty {
	type?: string;
	title?: string;
	description?: string;
	format?: string;
	writeOnly?: boolean;
	enum?: string[];
	default?: unknown;
	maxLength?: number;
}

/** A JSON Schema document, draft 2020-12. */
export interface SettingsSchema {
	properties?: Record<string, SchemaProperty>;
	required?: string[];
}

/**
 * The value that a secret field carries in place of its own. The hub returns it when the
 * field holds a value, and a save that names it keeps that value.
 */
export const SECRET_MASK = '******';

export type SettingsValue = string | boolean;

export type SettingsValues = Record<string, SettingsValue>;

export type SettingsInput = Record<string, string | boolean | null>;

/** The body that the hub answers with when a save does not match the schema. */
export interface ValidationFailure {
	reason: string;
	fields: Record<string, string>;
}
