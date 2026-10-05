import type { UiSchemaRoot } from '@sjsf/form';

import { SECRET_MASK, type SettingsInput, type SettingsValues } from '$lib/api';

import type { SettingsField } from './schema';

const SECRET_HELP_SET = 'A value is stored. Clear the field to remove it.';
const SECRET_HELP_UNSET = 'No value is stored.';
// One row holds the value of a person for each plugin, so every workspace reads it.
const PERSONAL_HELP = 'Personal. This value serves every workspace of yours.';

/**
 * Tells the form how to render a field that the schema alone does not describe. The
 * standard gives a secret `"format": "password"`, and no input type follows from it.
 * The standard gives no scope either, so a field of the person carries its own mark.
 */
export function uiSchemaFor(fields: SettingsField[], values: SettingsValues): UiSchemaRoot {
	const ui: UiSchemaRoot = {};

	for (const field of fields) {
		const personal = field.scope === 'user';
		const secret = field.kind === 'secret';

		if (!personal && !secret) {
			continue;
		}

		const help = [
			personal ? PERSONAL_HELP : undefined,
			secret ? (values[field.key] === SECRET_MASK ? SECRET_HELP_SET : SECRET_HELP_UNSET) : undefined
		].filter((line) => line !== undefined);

		ui[field.key] = {
			'ui:options': {
				...(personal ? { title: `${field.label} (personal)` } : {}),
				...(secret ? { text: { type: 'password', autocomplete: 'off' } } : {}),
				help: help.join(' ')
			}
		};
	}

	return ui;
}

/**
 * Builds the body of the save. A field that the form left empty travels as null, because
 * a field that the body does not name keeps its stored value.
 */
export function toInput(fields: SettingsField[], value: SettingsValues): SettingsInput {
	const input: SettingsInput = {};

	for (const field of fields) {
		const given = value[field.key];

		if (field.kind === 'switch') {
			input[field.key] = given === true;

			continue;
		}

		input[field.key] = typeof given === 'string' && given !== '' ? given : null;
	}

	return input;
}
