import { auth } from './auth';
import { plugins } from './plugins';

export { ApiError } from './errors';
export { buildAuthUrl } from './client';
export type { User, Plugin, UIManifest, UIRoute, RequestOptions } from './types';

export const api = {
	auth,
	plugins
};
