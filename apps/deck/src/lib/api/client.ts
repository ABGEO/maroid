import { env } from '$env/dynamic/public';

import { createClient, type ApiClient } from '@maroid/api-client';

const BASE_URL = (env.PUBLIC_HUB_BASE_URL ?? '').replace(/\/+$/, '');

let isRedirecting = false;

function landingUrl(): string {
	return `${window.location.origin}/auth/callback`;
}

function profileUrl(): string {
	return `${window.location.origin}/profile`;
}

export function signedOutUrl(): string {
	return `${window.location.origin}/signed-out`;
}

export function buildAuthUrl(): string {
	return `${BASE_URL}/auth?redirect=${encodeURIComponent(landingUrl())}`;
}

export function buildInviteUrl(token: string): string {
	const query = new URLSearchParams({ token, redirect: landingUrl() });
	return `${BASE_URL}/auth/invite?${query.toString()}`;
}

export function buildLinkUrl(provider: string): string {
	const query = new URLSearchParams({ provider, redirect: profileUrl() });
	return `${BASE_URL}/auth/link?${query.toString()}`;
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
