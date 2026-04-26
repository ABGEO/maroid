export interface User {
	name: string;
	picture: string;
}

export interface UIRoute {
	path: string;
	label: string;
}

export interface UIManifest {
	name: string;
	routes: UIRoute[];
}

export interface Plugin {
	id: string;
	version: string;
	ui?: UIManifest;
}

export interface RequestOptions {
	params?: Record<string, string | number | null | undefined>;
	headers?: HeadersInit;
	signal?: AbortSignal;
}
