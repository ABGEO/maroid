import { ApiError, PROBLEM_TYPE, api } from '$lib/api';

/**
 * Turns a plugin on or off in a workspace. Answers null when the hub accepts the
 * change, and the message to show when it refuses.
 */
export async function switchPlugin(
	workspaceId: string,
	pluginId: string,
	on: boolean
): Promise<string | null> {
	try {
		if (on) {
			await api.enablements.enable(workspaceId, pluginId);
		} else {
			await api.enablements.disable(workspaceId, pluginId);
		}

		return null;
	} catch (error) {
		return failureMessage(error, 'The plugin stays as it was. Try again.');
	}
}

function failureMessage(error: unknown, fallback: string): string {
	if (error instanceof ApiError && error.problem !== null) {
		if (error.is(PROBLEM_TYPE.permissionDenied) && error.problem.detail) {
			return error.problem.detail;
		}

		return error.problem.errors?.[0]?.detail ?? error.problem.title;
	}

	return fallback;
}
