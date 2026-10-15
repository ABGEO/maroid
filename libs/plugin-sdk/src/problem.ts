import { ApiError, PROBLEM_TYPE } from '@maroid/api-client';

/** The member that a JSON Pointer of a field failure names, as a person reads it. */
function fieldName(pointer: string): string {
	return pointer.replace(/^#\//, '').replaceAll('_', ' ');
}

/**
 * The reason that a person reads for a failed call: the first field that the hub
 * refused, the detail of the problem, or the fallback for any other failure.
 */
export function failureMessage(error: unknown, fallback: string): string {
	if (!(error instanceof ApiError) || error.problem === null) {
		return fallback;
	}

	if (error.is(PROBLEM_TYPE.preconditionFailed)) {
		return 'Someone changed this record since you opened it. Reload it and try again.';
	}

	const [field] = error.fields;
	if (field) {
		return `The ${fieldName(field.pointer)} ${field.detail}.`;
	}

	return error.problem.detail ?? error.problem.title;
}
