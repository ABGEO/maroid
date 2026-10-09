<script lang="ts">
	import { resolve } from '$app/paths';

	import { ApiError, PROBLEM_TYPE, api, type EnabledPlugin, type Plugin } from '$lib/api';
	import PluginSummary from '$lib/components/plugins/PluginSummary.svelte';
	import Alert from '$lib/components/ui/Alert.svelte';
	import Section from '$lib/components/ui/Section.svelte';
	import SkeletonList from '$lib/components/ui/SkeletonList.svelte';
	import { PERMISSION, holds } from '$lib/permissions';
	import { displayNameOf, hasCapability } from '$lib/plugins/capabilities';
	import { fieldsOf, missingFields } from '$lib/settings/schema';
	import { loadEnabled, pluginState } from '$lib/state/plugins.svelte';
	import { isMember } from '$lib/state/workspaces.svelte';

	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

	const workspaceId = $derived(data.workspace.id);
	const canSwitch = $derived(holds(data.workspace, PERMISSION.pluginsWrite));
	// An administrator manages a workspace of another person and reads none of its
	// settings, so the page offers them no state of the settings and no link to them.
	const readsSettings = $derived(isMember(workspaceId));

	let status = $state<'loading' | 'ready' | 'error'>('loading');
	let enabled = $state<EnabledPlugin[]>([]);
	let incomplete = $state<Record<string, boolean>>({});
	let busy = $state<string | null>(null);
	let banner = $state<string | null>(null);

	const enabledIds = $derived(enabled.map((one) => one.plugin_id));

	const inUse = $derived(
		enabled.map((one) => ({
			id: one.plugin_id,
			name: displayNameOf(one.plugin, one.plugin_id),
			plugin: one.plugin
		}))
	);

	const toTurnOn = $derived(
		pluginState.plugins.filter((plugin) => !enabledIds.includes(plugin.id))
	);

	const catalogLoading = $derived(
		canSwitch && (pluginState.status === 'idle' || pluginState.status === 'loading')
	);

	function configurable(plugin: Plugin | undefined): plugin is Plugin {
		return readsSettings && plugin !== undefined && hasCapability(plugin, 'settings');
	}

	function failureMessage(error: unknown, fallback: string): string {
		if (error instanceof ApiError && error.problem !== null) {
			if (error.is(PROBLEM_TYPE.permissionDenied) && error.problem.detail) {
				return error.problem.detail;
			}

			return error.problem.errors?.[0]?.detail ?? error.problem.title;
		}

		return fallback;
	}

	async function check(workspace: string, plugin: Plugin): Promise<void> {
		try {
			const [schema, values] = await Promise.all([
				api.settings.schema(workspace, plugin.id),
				api.settings.read(workspace, plugin.id)
			]);

			if (schema === null || values === null) {
				return;
			}

			incomplete[plugin.id] = missingFields(fieldsOf(schema), values.value).length > 0;
		} catch (error) {
			console.error('Failed to read the settings of a plugin', error);
		}
	}

	async function load(): Promise<void> {
		status = 'loading';

		try {
			const list = await api.enablements.list(workspaceId);
			if (list === null) {
				return;
			}

			enabled = list;
			incomplete = {};
			status = 'ready';
		} catch (error) {
			console.error('Failed to load the plugins of the workspace', error);
			status = 'error';
		}
	}

	async function toggle(pluginId: string, on: boolean): Promise<void> {
		busy = pluginId;
		banner = null;

		try {
			if (on) {
				await api.enablements.enable(workspaceId, pluginId);
			} else {
				await api.enablements.disable(workspaceId, pluginId);
			}
		} catch (error) {
			banner = failureMessage(error, 'The plugin stays as it was. Try again.');
		} finally {
			busy = null;
			await Promise.all([load(), loadEnabled(workspaceId, true)]);
		}
	}

	$effect(() => {
		void workspaceId;
		void load();
	});

	$effect(() => {
		for (const row of inUse) {
			if (configurable(row.plugin)) {
				void check(workspaceId, row.plugin);
			}
		}
	});
</script>

<div class="max-w-3xl">
	<header class="page-header">
		<div>
			<h1 class="page-title">Plugins</h1>
			<p class="page-lead">
				The plugins that {data.workspace.name} uses. A plugin that you switch off keeps its records and
				its settings.
			</p>
		</div>
	</header>

	{#if banner}
		<Alert kind="error" class="mb-6"><span>{banner}</span></Alert>
	{/if}

	{#if status === 'loading'}
		<SkeletonList row="h-16" />
	{:else if status === 'error'}
		<Alert kind="error" class="mb-6"
			><span>The hub answered no plugin. Reload the page.</span></Alert
		>
	{:else}
		<Section title="In use">
			{#if inUse.length === 0}
				<div class="empty-state">
					{data.workspace.name} uses no plugin.
				</div>
			{:else}
				<ul class="list panel">
					{#each inUse as row (row.id)}
						<li class="list-row flex items-center gap-4">
							<PluginSummary id={row.id} plugin={row.plugin} detailed>
								{#snippet badges()}
									{#if readsSettings && incomplete[row.id]}
										<span class="badge badge-warning badge-xs">Needs attention</span>
									{/if}
								{/snippet}
							</PluginSummary>

							{#if configurable(row.plugin)}
								<a
									class="btn btn-ghost btn-sm"
									href={resolve('/(dashboard)/w/[workspace]/plugins/[plugin]/settings', {
										workspace: workspaceId,
										plugin: row.id
									})}
								>
									Configure
								</a>
							{/if}

							{#if canSwitch}
								<input
									type="checkbox"
									class="toggle toggle-primary toggle-sm"
									aria-label="Use {row.name} in {data.workspace.name}"
									checked
									disabled={busy !== null}
									onchange={(event) => toggle(row.id, event.currentTarget.checked)}
								/>
							{/if}
						</li>
					{/each}
				</ul>
			{/if}
		</Section>

		{#if canSwitch}
			<Section title="To turn on">
				{#if catalogLoading}
					<SkeletonList count={1} />
				{:else if pluginState.status === 'error'}
					<Alert kind="error">
						<span>The hub answered no plugin to turn on. Reload the page.</span>
					</Alert>
				{:else if toTurnOn.length === 0}
					<div class="empty-state">
						No other plugin is open to you. An administrator puts a plugin on your allowlist.
					</div>
				{:else}
					<ul class="list panel">
						{#each toTurnOn as plugin (plugin.id)}
							<li class="list-row flex items-center justify-between gap-3">
								<PluginSummary id={plugin.id} {plugin} />
								<input
									type="checkbox"
									class="toggle toggle-primary toggle-sm"
									aria-label="Use {displayNameOf(plugin, plugin.id)} in {data.workspace.name}"
									checked={false}
									disabled={busy !== null}
									onchange={(event) => toggle(plugin.id, event.currentTarget.checked)}
								/>
							</li>
						{/each}
					</ul>
				{/if}
			</Section>
		{/if}
	{/if}
</div>
