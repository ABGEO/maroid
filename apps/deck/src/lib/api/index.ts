import { auth } from './auth';

export { ApiError } from './errors';
export { buildAuthUrl } from './client';
export type { User, RequestOptions } from './types';

export const api = {
	auth
};
