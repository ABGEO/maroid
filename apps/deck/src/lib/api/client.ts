import { env } from '$env/dynamic/public';

import { createClient, type ApiClient } from '@maroid/api-client';

import type { Handoff } from './types';

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

/**
 * Routes that start a flow at the identity provider answer 202 with
 * the address to visit, because a POST cannot redirect a browser that reached
 * it with fetch.
 */
async function startFlow(path: string, params: Record<string, string>): Promise<string> {
	const handoff = await client.post<Handoff>(path, undefined, { params });
	if (!handoff) {
		throw new Error('the hub named no authorization address');
	}

	return handoff.authorization_url;
}

export function startSignIn(): Promise<string> {
	return startFlow('/auth/sessions', { redirect: landingUrl() });
}

export function startInvitationRedemption(token: string): Promise<string> {
	return startFlow('/auth/invitation-redemptions', { token, redirect: landingUrl() });
}

export function startAttach(provider: string): Promise<string> {
	return startFlow('/auth/identities', { provider, redirect: profileUrl() });
}

function redirectToAuth(): void {
	if (isRedirecting) {
		return;
	}

	isRedirecting = true;
	void startSignIn().then((address) => {
		window.location.href = address;
	});
}

export const client: ApiClient = createClient({
	baseUrl: BASE_URL,
	onUnauthorized: redirectToAuth
});

export function createPluginClient(pluginId: string): ApiClient {
	return client.scope(`/plugins/${encodeURIComponent(pluginId)}/api`);
}
