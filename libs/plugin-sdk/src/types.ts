import type { ApiClient } from '@maroid/api-client';

export interface User {
	name: string;
	picture: string;
}

export interface PluginHost {
	user: User | null;
	/** Client already scoped to the prefix /workspaces/{workspaceId}/plugins/{id}/api. */
	api: ApiClient;
	/** Whether the role of the person holds a permission of this plugin, named without its prefix. */
	can(permission: string): boolean;
	/** Turn a plugin-relative path into a deck URL. */
	href(path: string): string;
	/** Client-side navigation to a plugin-relative path. */
	navigate(path: string): void;
	/** Name the page in the last link of the trail. The deck clears it when the page unmounts. */
	setTitle(title: string): void;
}

/** The value of each parameter segment of the matched page route, by its name. */
export type RouteParams = Record<string, string>;

export type RouteCleanup = () => void;

export type RouteMounter = (target: HTMLElement, props?: Record<string, unknown>) => RouteCleanup;
