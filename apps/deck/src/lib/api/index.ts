import { auth } from './auth';
import { enablements } from './enablements';
import { identities } from './identities';
import { plugins } from './plugins';
import { providers } from './providers';
import { settings } from './settings';
import { users } from './users';
import { workspaces } from './workspaces';

export {
	ApiError,
	FLOW_ID_HEADER,
	PROBLEM_MEDIA_TYPE,
	PROBLEM_TYPE,
	isProblem
} from '@maroid/api-client';
export type { ApiClient, FieldFailure, Problem, RequestOptions } from '@maroid/api-client';
export { createWriteIntent } from '@maroid/api-client';
export { createPluginClient, signedOutUrl, startSignIn } from './client';
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
	Workspace,
	Member,
	Role,
	Candidate,
	UserRecord,
	UserStatus,
	Invitation,
	InvitedUser,
	PluginRef,
	EnabledPlugin,
	InstanceWorkspace,
	UserIdentity,
	UserRef,
	Provider,
	ProviderPreset
} from './types';
export type { NewUser, SelfChange, UserChange } from './users';
export { LOCAL_PROVIDER } from './users';
export type { NewProvider, ProviderChange } from './providers';
export type { Tagged } from '@maroid/api-client';

export const api = {
	auth,
	enablements,
	identities,
	plugins,
	providers,
	settings,
	users,
	workspaces
};
