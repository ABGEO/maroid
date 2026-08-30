import { PUBLIC_HUB_BASE_URL } from '$env/static/public';

import { createClient, type ApiClient } from '@maroid/api-client';

const BASE_URL = PUBLIC_HUB_BASE_URL.replace(/\/+$/, '');

let isRedirecting = false;

export function buildAuthUrl(): string {
	const callbackUrl = `${window.location.origin}/auth/callback`;
	return `${BASE_URL}/auth?redirect=${encodeURIComponent(callbackUrl)}`;
}

function redirectToAuth(): void {
	if (isRedirecting) {
		return;
	}

	isRedirecting = true;
	window.location.href = buildAuthUrl();
}

export const client: ApiClient = createClient({
	baseUrl: BASE_URL,
	onUnauthorized: redirectToAuth
});

export function createPluginClient(pluginId: string): ApiClient {
	return client.scope(`/plugins/${encodeURIComponent(pluginId)}/api`);
}
