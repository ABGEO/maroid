import { PUBLIC_HUB_BASE_URL } from '$app/env/public';
import { navigating } from '$app/state';

import { createClient, type ApiClient } from '@maroid/api-client';
import type { FileClient, FileRecord, UploadTicket } from '@maroid/plugin-sdk';

import { RETURN_ADDRESS_PARAMETER, returnAddressOf } from '#lib/auth/return-address.ts';

import type { Handoff } from './types';

const BASE_URL = PUBLIC_HUB_BASE_URL.replace(/\/+$/, '');

let isRedirecting = false;

function landingUrl(returnAddress?: string): string {
	const landing = new URL('/auth/callback', window.location.origin);
	if (returnAddress) {
		landing.searchParams.set(RETURN_ADDRESS_PARAMETER, returnAddress);
	}

	return landing.toString();
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
		throw new Error('the server named no authorization address');
	}

	return handoff.authorization_url;
}

export function startSignIn(returnAddress?: string): Promise<string> {
	return startFlow('/auth/sessions', { redirect: landingUrl(returnAddress) });
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

	// The router runs the load of a route before it changes the address of the
	// window, so a 401 in a load fires while the window shows the previous page.
	const open = navigating.to?.url ?? new URL(window.location.href);
	void startSignIn(returnAddressOf(open)).then((address) => {
		window.location.href = address;
	});
}

export const client: ApiClient = createClient({
	baseUrl: BASE_URL,
	onUnauthorized: redirectToAuth
});

export function createPluginClient(workspaceId: string, pluginId: string): ApiClient {
	return client.scope(`${pluginPrefix(workspaceId, pluginId)}/api`);
}

function pluginPrefix(workspaceId: string, pluginId: string): string {
	return `/workspaces/${encodeURIComponent(workspaceId)}/plugins/${encodeURIComponent(pluginId)}`;
}

export function createFileClient(workspaceId: string, pluginId: string): FileClient {
	const plugin = client.scope(pluginPrefix(workspaceId, pluginId));

	return {
		async start(input) {
			const ticket = await plugin.post<UploadTicket>('/files', input);
			if (!ticket) {
				throw new Error('the hub answered no upload address');
			}

			return ticket;
		},

		async complete(id) {
			await plugin.post(`/files/${encodeURIComponent(id)}/completions`);
		},

		async get(id) {
			const file = await plugin.get<FileRecord>(`/files/${encodeURIComponent(id)}`);
			if (!file) {
				throw new Error('the hub answered no file');
			}

			return file;
		}
	};
}
