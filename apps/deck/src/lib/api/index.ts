import { auth } from './auth';
import { identities } from './identities';
import { plugins } from './plugins';
import { settings } from './settings';

export { ApiError } from '@maroid/api-client';
export type { ApiClient, RequestOptions } from '@maroid/api-client';
export { buildAuthUrl, createPluginClient, signedOutUrl } from './client';
export { SECRET_MASK } from './types';
export type {
	User,
	SignOut,
	Plugin,
	ProviderIdentity,
	UIManifest,
	UIRoute,
	SchemaProperty,
	SettingsSchema,
	SettingsValue,
	SettingsValues,
	SettingsInput,
	ValidationFailure
} from './types';

export const api = {
	auth,
	identities,
	plugins,
	settings
};
