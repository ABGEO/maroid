export interface User {
  name: string;
  picture: string;
}

export interface PluginHost {
  user: User | null;
}

export type RouteCleanup = () => void;

export type RouteMounter = (
  target: HTMLElement,
  props?: Record<string, unknown>,
) => RouteCleanup;
