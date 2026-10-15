import type { Capabilities, Plugin, UIManifest } from '$lib/api/types';

/** The name of one capability, as the hub reports it. */
export type CapabilityName = keyof Capabilities;

/**
 * The label of each capability, in the order that the plugins page shows them.
 */
const LABELS: Record<CapabilityName, string> = {
	ui: 'UI',
	widgets: 'Widgets',
	settings: 'Settings',
	api: 'API',
	telegram_commands: 'Bot commands',
	telegram_conversations: 'Conversations',
	mcp_tools: 'Agent tools',
	cron: 'Scheduled jobs',
	mqtt: 'MQTT',
	cli: 'CLI',
	permissions: 'Permissions',
	migrations: 'Migrations'
};

const ORDER = Object.keys(LABELS) as CapabilityName[];

/** Reports whether the hub loaded one capability for a plugin. */
export function hasCapability(plugin: Plugin, name: CapabilityName): boolean {
	return Boolean(plugin.capabilities?.[name]);
}

/** The user interface manifest of a plugin, when it declares one. */
export function uiOf(plugin: Plugin | undefined): UIManifest | undefined {
	return plugin?.capabilities?.ui;
}

/**
 * The name that the deck shows for a plugin. With no entry, the hub did not load the
 * plugin, and the whole identifier tells the person which one is missing.
 */
export function displayNameOf(plugin: Plugin | undefined, pluginId: string): string {
	return plugin?.name ?? pluginId;
}

/** The label of one capability. */
export function labelOf(name: CapabilityName): string {
	return LABELS[name] ?? name;
}

/**
 * How many items a capability holds. A capability that holds none answers zero,
 * so a caller shows the label alone.
 */
export function countOf(plugin: Plugin, name: CapabilityName): number {
	const value = plugin.capabilities?.[name];

	return Array.isArray(value) ? value.length : 0;
}

/** Every capability of a plugin, in the order that LABELS gives. */
export function capabilitiesOf(plugin: Plugin): CapabilityName[] {
	return ORDER.filter((name) => hasCapability(plugin, name));
}

/** One column of the table of a capability: a field of its items, and its heading. */
export interface Column {
	field: string;
	label: string;
	mono: boolean;
}

type ItemCapability = Exclude<CapabilityName, 'settings' | 'migrations' | 'ui'>;

const COLUMNS: Record<ItemCapability, Column[]> = {
	api: [
		{ field: 'method', label: 'Method', mono: true },
		{ field: 'path', label: 'Path', mono: true },
		{ field: 'permission', label: 'Permission', mono: true }
	],
	cli: [{ field: 'command', label: 'Command', mono: true }],
	cron: [
		{ field: 'id', label: 'Job', mono: true },
		{ field: 'schedule', label: 'Schedule', mono: true }
	],
	mqtt: [
		{ field: 'id', label: 'Subscriber', mono: true },
		{ field: 'topic', label: 'Topic', mono: true }
	],
	telegram_commands: [
		{ field: 'command', label: 'Command', mono: true },
		{ field: 'description', label: 'Description', mono: false },
		{ field: 'permission', label: 'Permission', mono: true }
	],
	telegram_conversations: [
		{ field: 'id', label: 'Conversation', mono: true },
		{ field: 'entry', label: 'Entry', mono: true }
	],
	mcp_tools: [
		{ field: 'name', label: 'Tool', mono: true },
		{ field: 'description', label: 'Description', mono: false },
		{ field: 'permission', label: 'Permission', mono: true }
	],
	widgets: [
		{ field: 'key', label: 'Key', mono: true },
		{ field: 'title', label: 'Title', mono: false },
		{ field: 'kind', label: 'Kind', mono: true },
		{ field: 'size', label: 'Size', mono: true },
		{ field: 'permission', label: 'Permission', mono: true }
	],
	permissions: [
		{ field: 'name', label: 'Permission', mono: true },
		{ field: 'role', label: 'Lowest role', mono: true }
	]
};

/**
 * The places where a long value may break: after each `/`, `.`, and `:`, so a
 * path or a permission wraps between its parts and never inside a word.
 */
export function partsOf(value: string): string[] {
	return value.split(/(?<=[/.:])/);
}

/**
 * The columns of the table of a capability. A capability that holds no item, and
 * the user interface, which lists its navigation entries, answer none.
 */
export function columnsOf(name: CapabilityName): Column[] {
	return name in COLUMNS ? COLUMNS[name as ItemCapability] : [];
}

/** The items of a capability, in the order of the report. */
export function itemsOf(plugin: Plugin, name: CapabilityName): object[] {
	const value = plugin.capabilities?.[name];

	return Array.isArray(value) ? value : [];
}

/** The value of one field of an item, as the report states it. */
export function cellOf(item: object, field: string): string {
	const value: unknown = (item as Record<string, unknown>)[field];

	return value === undefined || value === null ? '' : String(value);
}
