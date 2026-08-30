import type { ApiClient } from '@maroid/api-client';

export interface User {
  name: string;
  picture: string;
}

export interface PluginHost {
  user: User | null;
  /** Client already scoped to this plugin's /plugins/{id}/api prefix. */
  api: ApiClient;
  /** Turn a plugin-relative path into a deck URL. */
  href(path: string): string;
  /** Client-side navigation to a plugin-relative path. */
  navigate(path: string): void;
}

export type RouteCleanup = () => void;

export type RouteMounter = (
  target: HTMLElement,
  props?: Record<string, unknown>,
) => RouteCleanup;
