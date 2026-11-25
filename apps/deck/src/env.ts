import { defineEnvVars } from '@sveltejs/kit/env';

export const variables = defineEnvVars({
	PUBLIC_HUB_BASE_URL: {
		public: true,
		schema: (value) => value ?? '',
		description: 'The base URL of the hub API. Empty means the same origin as the deck.'
	}
});
