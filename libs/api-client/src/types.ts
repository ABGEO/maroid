export interface RequestOptions {
  params?: Record<string, string | number | null | undefined>;
  headers?: HeadersInit;
  signal?: AbortSignal;
}

export interface ClientConfig {
  /** Hub origin, e.g. https://hub.example.com. Trailing slashes are trimmed. */
  baseUrl: string;
  /** Path prepended to every request, e.g. /plugins/dev.maroid.jasmine/api. */
  prefix?: string;
  /** Called when the hub answers 401. The request then resolves to null. */
  onUnauthorized?: () => void;
  /** Injectable fetch, for tests and non-browser hosts. */
  fetch?: typeof globalThis.fetch;
}

export interface ApiClient {
  get<T>(path: string, options?: RequestOptions): Promise<T | null>;
  post<T>(path: string, body?: unknown, options?: RequestOptions): Promise<T | null>;
  put<T>(path: string, body?: unknown, options?: RequestOptions): Promise<T | null>;
  del<T>(path: string, options?: RequestOptions): Promise<T | null>;
  /** Derive a client with an additional path prefix, sharing this client's config. */
  scope(prefix: string): ApiClient;
}
