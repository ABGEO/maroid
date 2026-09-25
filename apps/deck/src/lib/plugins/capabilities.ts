import type { Capabilities, Plugin, UIManifest } from '$lib/api/types';

/** The name of one capability, as the hub reports it. */
export type CapabilityName = keyof Capabilities;

/**
 * The label of each capability, in the order that the plugins page shows them.
 */
const LABELS: Record<CapabilityName, string> = {
	ui: 'UI',
	settings: 'Settings',
	api: 'API',
	telegram_commands: 'Bot commands',
	telegram_conversations: 'Conversations',
	mcp_tools: 'Agent tools',
	cron: 'Scheduled jobs',
	mqtt: 'MQTT',
	cli: 'CLI',
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
 * The name that the deck shows for a plugin. The manifest name wins, and the
 * last part of the identifier answers when the plugin declares no user interface.
 */
export function displayNameOf(plugin: Plugin | undefined, pluginId: string): string {
	return uiOf(plugin)?.name ?? pluginId.split('.').pop() ?? pluginId;
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

/** Every capability of a plugin, in the order that LABELS gives. PCAP-FR-006. */
export function capabilitiesOf(plugin: Plugin): CapabilityName[] {
	return ORDER.filter((name) => hasCapability(plugin, name));
}
