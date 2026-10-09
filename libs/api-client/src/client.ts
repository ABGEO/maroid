import { ApiError } from './errors';
import { FLOW_ID_HEADER, PROBLEM_MEDIA_TYPE } from './problem';
import type { ApiClient, ClientConfig, LinkOptions, RequestOptions, Tagged } from './types';

const IDEMPOTENCY_KEY_HEADER = 'Idempotency-Key';

const MERGE_PATCH_MEDIA_TYPE = 'application/merge-patch+json';

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

  /** Answer the body of an exchange beside the entity tag of its answer. */
  function tagged<T>(exchanged: { body: T | null; response: Response } | null): Tagged<T> | null {
    if (exchanged === null) {
      return null;
    }

    return {
      value: exchanged.body as T,
      etag: exchanged.response.headers.get('ETag') ?? undefined
    };
  }

  /** Build the exchange of a JSON Merge Patch, RFC 7396. */
  function mergePatch(path: string, body: unknown, options: RequestOptions) {
    const headers = new Headers(options.headers);
    if (!headers.has('Content-Type')) {
      headers.set('Content-Type', MERGE_PATCH_MEDIA_TYPE);
    }

    return {
      url: buildUrl(path, options.params),
      init: {
        method: 'PATCH',
        body: serializeBody(body),
        headers,
        signal: options.signal
      } satisfies RequestInit
    };
  }

  function resolveLink(link: string): string {
    const hub = new URL(baseUrl || globalThis.location.origin);
    const target = new URL(link, hub);

    if (target.origin !== hub.origin) {
      throw new Error(`the link ${link} leaves the origin of the hub`);
    }

    return target.href;
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
      return tagged(
        await exchange<T>(buildUrl(path, options.params), {
          method: 'GET',
          headers: options.headers,
          signal: options.signal
        })
      );
    },

    follow<T>(link: string, options: LinkOptions = {}) {
      return request<T>(resolveLink(link), {
        method: 'GET',
        headers: options.headers,
        signal: options.signal
      });
    },

    post<T>(path: string, body?: unknown, options: RequestOptions = {}) {
      const headers = new Headers(options.headers);
      if (options.idempotencyKey !== undefined && !headers.has(IDEMPOTENCY_KEY_HEADER)) {
        headers.set(IDEMPOTENCY_KEY_HEADER, options.idempotencyKey);
      }

      return request<T>(buildUrl(path, options.params), {
        method: 'POST',
        body: serializeBody(body),
        headers,
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

    patch<T>(path: string, body: unknown, options: RequestOptions = {}) {
      const { url, init } = mergePatch(path, body, options);

      return request<T>(url, init, options.ifMatch);
    },

    async patchTagged<T>(path: string, body: unknown, options: RequestOptions = {}) {
      const { url, init } = mergePatch(path, body, options);

      return tagged(await exchange<T>(url, init, options.ifMatch));
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
