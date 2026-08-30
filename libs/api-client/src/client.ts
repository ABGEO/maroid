import { ApiError } from './errors';
import type { ApiClient, ClientConfig, RequestOptions } from './types';

function trimTrailingSlashes(value: string): string {
  return value.replace(/\/+$/, '');
}

function normalizeSegment(value: string | undefined): string {
  if (!value) {
    return '';
  }

  const withLeadingSlash = value.startsWith('/') ? value : `/${value}`;

  return trimTrailingSlashes(withLeadingSlash);
}

export function createClient(config: ClientConfig): ApiClient {
  const baseUrl = trimTrailingSlashes(config.baseUrl);
  const prefix = normalizeSegment(config.prefix);
  const fetchImpl = config.fetch ?? globalThis.fetch.bind(globalThis);

  function buildUrl(path: string, params?: RequestOptions['params']): string {
    const normalizedPath = path.startsWith('/') ? path : `/${path}`;
    const url = `${baseUrl}${prefix}${normalizedPath}`;

    if (!params) {
      return url;
    }

    const qs = new URLSearchParams();
    for (const [key, value] of Object.entries(params)) {
      if (value !== null && value !== undefined) {
        qs.set(key, String(value));
      }
    }

    const queryString = qs.toString();

    return queryString ? `${url}?${queryString}` : url;
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

  async function request<T>(url: string, init: RequestInit): Promise<T | null> {
    const headers = new Headers(init.headers);
    if (!headers.has('Accept')) {
      headers.set('Accept', 'application/json');
    }

    if (init.body !== undefined && !headers.has('Content-Type')) {
      headers.set('Content-Type', 'application/json');
    }

    const response = await fetchImpl(url, { ...init, credentials: 'include', headers });
    const body = await parseBody(response);

    if (response.status === 401) {
      config.onUnauthorized?.();

      return null;
    }

    if (!response.ok) {
      throw new ApiError(response.status, response.statusText, body);
    }

    return body as T;
  }

  function serializeBody(body?: unknown): string | undefined {
    return body !== undefined ? JSON.stringify(body) : undefined;
  }

  const client: ApiClient = {
    get<T>(path: string, options: RequestOptions = {}) {
      return request<T>(buildUrl(path, options.params), {
        method: 'GET',
        headers: options.headers,
        signal: options.signal
      });
    },

    post<T>(path: string, body?: unknown, options: RequestOptions = {}) {
      return request<T>(buildUrl(path, options.params), {
        method: 'POST',
        body: serializeBody(body),
        headers: options.headers,
        signal: options.signal
      });
    },

    put<T>(path: string, body?: unknown, options: RequestOptions = {}) {
      return request<T>(buildUrl(path, options.params), {
        method: 'PUT',
        body: serializeBody(body),
        headers: options.headers,
        signal: options.signal
      });
    },

    del<T>(path: string, options: RequestOptions = {}) {
      return request<T>(buildUrl(path, options.params), {
        method: 'DELETE',
        headers: options.headers,
        signal: options.signal
      });
    },

    scope(childPrefix: string) {
      return createClient({ ...config, prefix: `${prefix}${normalizeSegment(childPrefix)}` });
    }
  };

  return client;
}
