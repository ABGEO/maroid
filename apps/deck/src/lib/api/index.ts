import { auth } from './auth';
import { plugins } from './plugins';

export { ApiError } from '@maroid/api-client';
export type { ApiClient, RequestOptions } from '@maroid/api-client';
export { buildAuthUrl, createPluginClient } from './client';
export type { User, Plugin, UIManifest, UIRoute } from './types';

export const api = {
	auth,
	plugins
};
