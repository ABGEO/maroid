export interface RequestOptions {
  params?: Record<string, string | number | null | undefined>;
  headers?: HeadersInit;
  signal?: AbortSignal;
  /**
   * The entity tag that an earlier read answered. The write lands only while
   * the record still holds it, and a record that moved answers 412. A write
   * that names none lands.
   */
  ifMatch?: string;
  /**
   * The key of a write that creates. The hub answers a repeat under one key
   * with the earlier result. `post` alone sends it.
   */
  idempotencyKey?: string;
}

/**
 * A body and the entity tag that came with it. A caller holds the tag and sends
 * it back on the write that guards this state.
 */
export interface Tagged<T> {
  value: T;
  /** Absent when the answer carried no entity tag. */
  etag?: string;
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
  /**
   * Read a record and the entity tag that guards it. The client keeps nothing
   * between this call and the write, so the caller holds the tag itself.
   */
  getTagged<T>(path: string, options?: RequestOptions): Promise<Tagged<T> | null>;
  post<T>(path: string, body?: unknown, options?: RequestOptions): Promise<T | null>;
  put<T>(path: string, body?: unknown, options?: RequestOptions): Promise<T | null>;
  del<T>(path: string, options?: RequestOptions): Promise<T | null>;
  /** Derive a client with an additional path prefix, sharing this client's config. */
  scope(prefix: string): ApiClient;
}
