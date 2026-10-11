import type { FieldFailure, NotificationProviderKind, SettingsValues } from '$lib/api';
import { fieldsOf } from '$lib/settings/schema';

/**
 * Keeps the failures that name a field of one member, and moves each pointer under that
 * member, so the form of the member places the message on its own field. The hub names
 * a field of a provider as `#/fields/<key>` and a field of a destination as
 * `#/destination/<key>`.
 */
export function failuresUnder(failures: FieldFailure[], member: string): FieldFailure[] {
	const prefix = `#/${member}/`;

	return failures
		.filter((failure) => failure.pointer.startsWith(prefix))
		.map((failure) => ({ ...failure, pointer: `#/${failure.pointer.slice(prefix.length)}` }));
}

/** Writes a destination as `Label value` pairs, in the order that the kind declares. */
export function describeDestination(
	destination: SettingsValues,
	kind: NotificationProviderKind | undefined
): string {
	if (kind === undefined) {
		return Object.values(destination).join(', ');
	}

	return fieldsOf(kind.destination_schema)
		.filter((field) => typeof destination[field.key] === 'string' && destination[field.key] !== '')
		.map((field) => `${field.label} ${String(destination[field.key])}`)
		.join(', ');
}
