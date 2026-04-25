export interface User {
	name: string;
	picture: string;
}

export interface RequestOptions {
	params?: Record<string, string | number | null | undefined>;
	headers?: HeadersInit;
	signal?: AbortSignal;
}
