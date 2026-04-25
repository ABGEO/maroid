import { PUBLIC_HUB_BASE_URL } from '$env/static/public';

import { ApiError } from './errors';
import type { RequestOptions } from './types';

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

function buildUrl(
	path: string,
	params?: Record<string, string | number | null | undefined>
): string {
	const normalizedPath = path.startsWith('/') ? path : `/${path}`;
	const base = `${BASE_URL}${normalizedPath}`;

	if (!params) {
		return base;
	}

	const qs = new URLSearchParams();
	for (const [key, value] of Object.entries(params)) {
		if (value !== null && value !== undefined) {
			qs.set(key, String(value));
		}
	}

	const queryString = qs.toString();
	return queryString ? `${base}?${queryString}` : base;
}

async function request<T>(path: string, init: RequestInit = {}): Promise<T | null> {
	const headers = new Headers(init.headers);
	if (!headers.has('Accept')) {
		headers.set('Accept', 'application/json');
	}

	if (init.body !== undefined && !headers.has('Content-Type')) {
		headers.set('Content-Type', 'application/json');
	}

	const response = await fetch(path, {
		...init,
		credentials: 'include',
		headers
	});

	const body = await parseBody(response);

	if (response.status === 401) {
		redirectToAuth();

		return null;
	}

	if (!response.ok) {
		throw new ApiError(response.status, response.statusText, body);
	}

	return body as T;
}

async function parseBody(response: Response): Promise<unknown> {
	if (response.status === 204) {
		return null;
	}

	const contentType = response.headers.get('Content-Type') ?? '';
	if (contentType.includes('application/json')) {
		return response.json();
	}

	const text = await response.text();
	return text.length > 0 ? text : null;
}

export function get<T>(path: string, options: RequestOptions = {}): Promise<T | null> {
	const { params, headers, signal } = options;
	return request<T>(buildUrl(path, params), { method: 'GET', headers, signal });
}

export function post<T>(
	path: string,
	body?: unknown,
	options: RequestOptions = {}
): Promise<T | null> {
	const { headers, signal } = options;
	return request<T>(buildUrl(path), {
		method: 'POST',
		body: bodyOrNull(body),
		headers,
		signal
	});
}

function bodyOrNull(body?: unknown): string | undefined {
	if (body !== undefined) {
		return JSON.stringify(body);
	}

	return undefined;
}
