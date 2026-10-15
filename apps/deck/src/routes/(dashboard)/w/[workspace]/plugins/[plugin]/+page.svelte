<script lang="ts">
	import { goto, invalidate } from '$app/navigation';
	import { resolve } from '$app/paths';

	import CapabilityPanel from '$lib/components/plugins/CapabilityPanel.svelte';
	import Alert from '$lib/components/ui/Alert.svelte';
	import Section from '$lib/components/ui/Section.svelte';
	import { PERMISSION, holds } from '$lib/permissions';
	import { capabilitiesOf, displayNameOf, hasCapability } from '$lib/plugins/capabilities';
	import { switchPlugin } from '$lib/plugins/switch';
	import { canTurnOn, loadEnabled, pluginState } from '$lib/state/plugins.svelte';
	import { isMember } from '$lib/state/workspaces.svelte';

	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

	const workspaceId = $derived(data.workspace.id);
	const inUse = $derived(data.enablement !== null);
	const entry = $derived(
		data.enablement?.plugin ?? pluginState.plugins.find((plugin) => plugin.id === data.pluginId)
	);
	const name = $derived(displayNameOf(entry, data.pluginId));
	const capabilities = $derived(entry ? capabilitiesOf(entry) : []);
	const canSwitch = $derived(holds(data.workspace, PERMISSION.pluginsWrite));
	// An administrator manages a workspace of another person and reads none of its
	// settings, as on the list of the plugins.
	const configurable = $derived(
		inUse && entry !== undefined && hasCapability(entry, 'settings') && isMember(workspaceId)
	);

	let busy = $state(false);
	let banner = $state<string | null>(null);

	function entryHref(path: string) {
		return resolve('/(dashboard)/w/[workspace]/plugins/[plugin]/[...path]', {
			workspace: workspaceId,
			plugin: data.pluginId,
			path: path.replace(/^\/+|\/+$/g, '')
		});
	}

	async function toggle(input: HTMLInputElement): Promise<void> {
		const on = input.checked;
		busy = true;
		banner = null;
		banner = await switchPlugin(workspaceId, data.pluginId, on);

		if (!on && banner === null && !(await canTurnOn(data.pluginId))) {
			await loadEnabled(workspaceId, true);
			await goto(resolve('/(dashboard)/w/[workspace]/plugins', { workspace: workspaceId }));

			return;
		}

		await Promise.all([invalidate('app:plugin-detail'), loadEnabled(workspaceId, true)]);
		input.checked = inUse;
		busy = false;
	}
</script>

<div class="max-w-4xl">
	<header class="page-header">
		<div class="min-w-0">
			<div class="flex items-center gap-2">
				<h1 class="page-title">{name}</h1>
				{#if entry}
					<span class="badge badge-ghost badge-sm font-mono">v{entry.version}</span>
				{/if}
			</div>
			<p class="meta mt-1 font-mono">{data.pluginId}</p>
			{#if entry?.description}
				<p class="page-lead">{entry.description}</p>
			{/if}
			<p class="mt-2 text-sm">
				{inUse ? 'In use' : 'Not in use'} in {data.workspace.name}.
			</p>
		</div>

		{#if configurable || canSwitch}
			<div class="flex items-center gap-3">
				{#if configurable}
					<a
						class="btn btn-ghost btn-sm"
						href={resolve('/(dashboard)/w/[workspace]/plugins/[plugin]/settings', {
							workspace: workspaceId,
							plugin: data.pluginId
						})}
					>
						Configure
					</a>
				{/if}

				{#if canSwitch}
					<input
						type="checkbox"
						class="toggle toggle-primary toggle-sm"
						aria-label="Use {name} in {data.workspace.name}"
						checked={inUse}
						disabled={busy}
						onchange={(event) => toggle(event.currentTarget)}
					/>
				{/if}
			</div>
		{/if}
	</header>

	{#if banner}
		<Alert kind="error" class="mb-6"><span>{banner}</span></Alert>
	{/if}

	{#if inUse && !entry}
		<Alert kind="info" class="mb-6">
			<span>The hub did not load this plugin. Its records and its settings stay.</span>
		</Alert>
	{/if}

	{#if entry}
		<Section title="Capabilities">
			{#if capabilities.length === 0}
				<div class="empty-state">This plugin declares no capability.</div>
			{:else}
				<div class="flex flex-col gap-4">
					{#each capabilities as capability (capability)}
						<CapabilityPanel
							plugin={entry}
							name={capability}
							entryHref={inUse ? entryHref : undefined}
						/>
					{/each}
				</div>
			{/if}
		</Section>
	{/if}
</div>
