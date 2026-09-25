import { auth } from './auth';
import { identities } from './identities';
import { plugins } from './plugins';
import { settings } from './settings';

export {
	ApiError,
	FLOW_ID_HEADER,
	PROBLEM_MEDIA_TYPE,
	PROBLEM_TYPE,
	isProblem
} from '@maroid/api-client';
export type { ApiClient, FieldFailure, Problem, RequestOptions } from '@maroid/api-client';
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
	SettingsInput
} from './types';

export const api = {
	auth,
	identities,
	plugins,
	settings
};
