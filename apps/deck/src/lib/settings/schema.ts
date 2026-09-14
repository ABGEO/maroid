import type { SchemaProperty, SettingsSchema, SettingsValue } from '$lib/api';

export type FieldKind = 'text' | 'secret' | 'switch' | 'choice';

export interface SettingsField {
	key: string;
	kind: FieldKind;
	label: string;
	description?: string;
	choices: string[];
	maxLength?: number;
	required: boolean;
}

// The hub reads the kind of a field from these keywords, in this order.
function kindOf(property: SchemaProperty): FieldKind {
	if (property.writeOnly === true || property.format === 'password') {
		return 'secret';
	}

	if (property.enum !== undefined && property.enum.length > 0) {
		return 'choice';
	}

	if (property.type === 'boolean') {
		return 'switch';
	}

	return 'text';
}

/** Reads the fields of a settings schema, in the order that the plugin declared them. */
export function fieldsOf(schema: SettingsSchema): SettingsField[] {
	const required = new Set(schema.required ?? []);

	return Object.entries(schema.properties ?? {}).map(([key, property]) => ({
		key,
		kind: kindOf(property),
		label: property.title ?? key,
		description: property.description,
		choices: property.enum ?? [],
		maxLength: property.maxLength,
		required: required.has(key)
	}));
}

/**
 * Reports whether the hub holds a stored value for the field. A secret carries the mask
 * when it holds one and the empty string when it holds none.
 */
export function holdsValue(field: SettingsField, values: Record<string, SettingsValue>): boolean {
	const value = values[field.key];

	if (field.kind === 'switch') {
		return typeof value === 'boolean';
	}

	return typeof value === 'string' && value !== '';
}

/**
 * Names each required field that holds no value. The hub reports the settings of
 * such a plugin as absent, and its job ends every run.
 */
export function missingFields(
	fields: SettingsField[],
	values: Record<string, SettingsValue>
): string[] {
	return fields.filter((field) => field.required && !holdsValue(field, values)).map((f) => f.key);
}
