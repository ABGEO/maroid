export interface User {
	first_name?: string;
	last_name?: string;
	picture: string;
	/** The connector that authenticated this session. */
	provider: string;
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
	name: string;
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
