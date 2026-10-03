import { error } from '@sveltejs/kit';

import { ApiError, api } from '$lib/api';

import type { LayoutLoad } from './$types';

export const load: LayoutLoad = async ({ params }) => {
	try {
		const tagged = await api.workspaces.get(params.workspace);
		if (tagged === null) {
			error(404, 'No workspace of yours has this address.');
		}

		return { workspace: tagged.value, etag: tagged.etag };
	} catch (failure) {
		if (failure instanceof ApiError && failure.status === 404) {
			error(404, 'No workspace of yours has this address.');
		}

		throw failure;
	}
};
