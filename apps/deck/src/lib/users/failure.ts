import { ApiError, PROBLEM_TYPE } from '$lib/api';

/** Answers the sentence that a failed write of a user record shows to the administrator. */
export function userFailureMessage(error: unknown, fallback: string): string {
	if (!(error instanceof ApiError) || error.problem === null) {
		return fallback;
	}

	if (error.is(PROBLEM_TYPE.administratorLast)) {
		return 'The instance keeps one active administrator. Mark another user first.';
	}

	if (error.is(PROBLEM_TYPE.administratorAllowlist)) {
		return 'An administrator reaches every plugin and holds no allowlist.';
	}

	return error.problem.errors?.[0]?.detail ?? error.problem.title;
}
