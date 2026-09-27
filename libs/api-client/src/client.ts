import { ApiError } from './errors';
import { FLOW_ID_HEADER, PROBLEM_MEDIA_TYPE } from './problem';
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

  function isJson(contentType: string): boolean {
    const mediaType = contentType.split(';', 1)[0].trim().toLowerCase();

    return mediaType === 'application/json' || mediaType.endsWith('+json');
  }

  async function parseBody(response: Response): Promise<unknown> {
    if (response.status === 204) {
      return null;
    }

    if (isJson(response.headers.get('Content-Type') ?? '')) {
      return response.json();
    }

    const text = await response.text();

    return text.length > 0 ? text : null;
  }

  /**
   * Send one request and answer the body beside the response, so that a caller
   * can read a header of a successful answer. `request` drops the response for
   * every caller that needs only the body.
   */
  async function exchange<T>(
    url: string,
    init: RequestInit,
    ifMatch?: string
  ): Promise<{ body: T | null; response: Response } | null> {
    const headers = new Headers(init.headers);
    if (!headers.has('Accept')) {
      headers.set('Accept', `application/json, ${PROBLEM_MEDIA_TYPE}`);
    }

    if (init.body !== undefined && !headers.has('Content-Type')) {
      headers.set('Content-Type', 'application/json');
    }

    if (ifMatch !== undefined && !headers.has('If-Match')) {
      headers.set('If-Match', ifMatch);
    }

    const response = await fetchImpl(url, {
      ...init,
      credentials: 'include',
      headers
    });
    const body = await parseBody(response);

    if (response.status === 401) {
      config.onUnauthorized?.();

      return null;
    }

    if (!response.ok) {
      throw new ApiError(
        response.status,
        response.statusText,
        body,
        response.headers.get(FLOW_ID_HEADER) ?? ''
      );
    }

    return { body: body as T | null, response };
  }

  async function request<T>(url: string, init: RequestInit, ifMatch?: string): Promise<T | null> {
    const exchanged = await exchange<T>(url, init, ifMatch);

    return exchanged?.body ?? null;
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

    async getTagged<T>(path: string, options: RequestOptions = {}) {
      const exchanged = await exchange<T>(buildUrl(path, options.params), {
        method: 'GET',
        headers: options.headers,
        signal: options.signal
      });

      if (exchanged === null) {
        return null;
      }

      return {
        value: exchanged.body as T,
        etag: exchanged.response.headers.get('ETag') ?? undefined
      };
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
      return request<T>(
        buildUrl(path, options.params),
        {
          method: 'PUT',
          body: serializeBody(body),
          headers: options.headers,
          signal: options.signal
        },
        options.ifMatch
      );
    },

    del<T>(path: string, options: RequestOptions = {}) {
      return request<T>(
        buildUrl(path, options.params),
        {
          method: 'DELETE',
          headers: options.headers,
          signal: options.signal
        },
        options.ifMatch
      );
    },

    scope(childPrefix: string) {
      return createClient({
        ...config,
        prefix: `${prefix}${normalizeSegment(childPrefix)}`
      });
    }
  };

  return client;
}
