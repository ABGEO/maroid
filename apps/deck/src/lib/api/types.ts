export interface User {
	/** The identifier of the user record. */
	id: string;
	first_name?: string;
	last_name?: string;
	picture?: string;
	/** The connector that authenticated this session. */
	provider: string;
	/** Whether the person administers the instance. */
	is_administrator: boolean;
}

export type UserStatus = 'active' | 'blocked';

/** A user record, as an administrator reads it. */
export interface UserRecord {
	id: string;
	first_name?: string;
	last_name?: string;
	status: UserStatus;
	is_administrator: boolean;
	created_at: string;
	updated_at: string;
}

/** One identity of a user record, as an administrator reads it. */
export interface UserIdentity {
	provider: string;
	/** The email address of a local account, or the handle at the provider. */
	username?: string;
	display_name?: string;
	created_at: string;
}

/** A user record that the report of a removal names. */
export interface UserRef {
	id: string;
	first_name?: string;
	last_name?: string;
}

export type ProviderPreset = 'local' | 'telegram' | 'oidc';

/** One provider of the instance. It never carries a secret. */
export interface Provider {
	id: string;
	/** The label of the button on the sign in page of Dex. */
	name: string;
	/** Absent on a static provider, which the file of Dex holds. */
	preset?: ProviderPreset;
	static: boolean;
	issuer?: string;
	client_id?: string;
	client_secret_set?: boolean;
	user_id_key?: string;
	scopes?: string[];
	options?: Record<string, unknown>;
	/** The address to register at the upstream provider. */
	redirect_uri?: string;
	/** The identities that a removal deletes. */
	identity_count: number;
	/** The administrators who hold no sign in after a removal. */
	administrators_without_sign_in: UserRef[];
}

/** The address that redeems an invitation. The hub answers it one time. */
export interface Invitation {
	address: string;
	expires_at: string;
}

/** A new user record and its first invitation. */
export interface InvitedUser {
	user: UserRecord;
	invitation: Invitation;
}

/** A plugin on an allowlist, or a plugin that a workspace enables. */
export interface PluginRef {
	plugin_id: string;
	created_at: string;
}

/**
 * A plugin that a workspace enables. `plugin` is its entry, absent when the hub did not
 * load the plugin. Every member reads it, whatever their allowlist holds.
 */
export interface EnabledPlugin {
	plugin_id: string;
	created_at: string;
	plugin?: Plugin;
}

/** A workspace of the instance, as an administrator reads it. */
export interface InstanceWorkspace {
	id: string;
	name: string;
	member_count: number;
	plugin_ids: string[];
	created_at: string;
	updated_at: string;
}

/**
 * The body that a route starting a flow answers. It names the address of the
 * identity provider that the browser visits next.
 */
export interface Handoff {
	authorization_url: string;
}

/** The body of a sign out. It names the target that the browser goes to. */
export interface SignOut {
	redirect: string;
}

export interface UIRoute {
	path: string;
	label: string;
}

export interface UIManifest {
	routes: UIRoute[];
}

/** One route that a plugin serves, with the prefix that the hub mounts it under. */
export interface APIRoute {
	method: string;
	path: string;
}

/** One bot command that a plugin answers, with the prefix of the plugin. */
export interface TelegramCommand {
	command: string;
	description: string;
}

/** One tool that a plugin exposes over the Model Context Protocol. */
export interface MCPTool {
	name: string;
	description: string;
}

/** One job that a plugin runs on a schedule. */
export interface CronJob {
	id: string;
	schedule: string;
}

/** One topic that a plugin subscribes to. */
export interface MQTTSubscriber {
	id: string;
	topic: string;
}

/** One conversation that a plugin drives. */
export interface TelegramConversation {
	id: string;
	entry: string;
}

/** One command that a plugin adds to the command tree. */
export interface CLICommand {
	command: string;
}

/**
 * What a plugin can do. A key is present when the hub loaded that capability,
 * and absent otherwise. A capability that holds no item carries `true`, so every
 * present capability is truthy.
 */
export interface Capabilities {
	settings?: true;
	migrations?: true;
	ui?: UIManifest;
	api?: APIRoute[];
	cli?: CLICommand[];
	cron?: CronJob[];
	mqtt?: MQTTSubscriber[];
	telegram_commands?: TelegramCommand[];
	telegram_conversations?: TelegramConversation[];
	mcp_tools?: MCPTool[];
}

export interface Plugin {
	id: string;
	name: string;
	description?: string;
	version: string;
	capabilities: Capabilities;
}

/** One provider that Maroid offers, attached to the acting user's record or not. */
export interface ProviderIdentity {
	provider: string;
	name: string;
	attached: boolean;
	username?: string;
	display_name?: string;
	picture_url?: string;
	attached_at?: string;
}

/** One property of a settings schema. The hub infers it from the struct of the plugin. */
export interface SchemaProperty {
	type?: string;
	title?: string;
	description?: string;
	format?: string;
	writeOnly?: boolean;
	enum?: string[];
	default?: unknown;
	maxLength?: number;
	'x-maroid-scope'?: 'workspace' | 'user';
}

/** A JSON Schema document, draft 2020-12. */
export interface SettingsSchema {
	properties?: Record<string, SchemaProperty>;
	required?: string[];
}

/**
 * The value that a secret field carries in place of its own. The hub returns it when the
 * field holds a value, and a save that names it keeps that value.
 */
export const SECRET_MASK = '******';

export type SettingsValue = string | boolean;

export type SettingsValues = Record<string, SettingsValue>;

export type SettingsInput = Record<string, string | boolean | null>;

/** A workspace that the person is a member of. */
/** The role of a member in a workspace, from the most access to the least. */
export type Role = 'manager' | 'editor' | 'viewer';

export interface Workspace {
	id: string;
	name: string;
	/** The role of the person in the workspace. */
	role: Role;
	/** Every permission that the role holds. The read of one workspace answers it. */
	permissions?: string[];
	created_at: string;
	updated_at: string;
}

/** One member of a workspace. A name is absent when the record holds none. */
export interface Member {
	user_id: string;
	first_name?: string;
	last_name?: string;
	role: Role;
	created_at: string;
	updated_at: string;
}

/** An active user record that is no member of the workspace yet. */
export interface Candidate {
	user_id: string;
	first_name?: string;
	last_name?: string;
}
