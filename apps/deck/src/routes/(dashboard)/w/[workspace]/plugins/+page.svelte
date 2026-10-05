<script lang="ts">
	import { resolve } from '$app/paths';

	import {
		ApiError,
		PROBLEM_TYPE,
		api,
		isProblem,
		type EnabledPlugin,
		type Plugin
	} from '$lib/api';
	import { PERMISSION, holds } from '$lib/permissions';
	import {
		capabilitiesOf,
		countOf,
		displayNameOf,
		hasCapability,
		labelOf
	} from '$lib/plugins/capabilities';
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
		if (error instanceof ApiError && isProblem(error.body)) {
			if (error.body.type === PROBLEM_TYPE.permissionDenied && error.body.detail) {
				return error.body.detail;
			}

			return error.body.errors?.[0]?.detail ?? error.body.title;
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
	<h1 class="font-display text-[42px] leading-[1.05] tracking-tight">Plugins</h1>
	<p class="text-base-content/60 mt-3 text-sm">
		The plugins that {data.workspace.name} uses. A plugin that you switch off keeps its records and its
		settings.
	</p>

	{#if banner}
		<div class="alert alert-error mt-6">
			<span>{banner}</span>
		</div>
	{/if}

	{#if status === 'loading'}
		<div class="mt-8 flex flex-col gap-2" aria-hidden="true">
			{#each [0, 1, 2] as i (i)}
				<span class="skeleton h-16 w-full"></span>
			{/each}
		</div>
	{:else if status === 'error'}
		<div class="alert alert-error mt-6">
			<span>The hub answered no plugin. Reload the page.</span>
		</div>
	{:else}
		<h2 class="mt-8 text-sm font-semibold">In use</h2>

		{#if inUse.length === 0}
			<div class="border-base-300 text-base-content/60 mt-3 rounded-md border border-dashed p-6">
				{data.workspace.name} uses no plugin.
			</div>
		{:else}
			<ul class="border-base-300 rounded-box mt-3 divide-y border">
				{#each inUse as row (row.id)}
					<li class="flex items-center gap-4 px-4 py-3">
						<div class="min-w-0 flex-1">
							<div class="flex items-center gap-2">
								<span class="text-[15px] font-semibold">{row.name}</span>
								{#if row.plugin}
									<span class="badge badge-ghost badge-xs font-mono">v{row.plugin.version}</span>
								{/if}
								{#if readsSettings && incomplete[row.id]}
									<span class="badge badge-warning badge-xs">Needs attention</span>
								{/if}
							</div>
							<div class="text-base-content/50 truncate font-mono text-[11px]">{row.id}</div>
							{#if row.plugin && capabilitiesOf(row.plugin).length > 0}
								<div class="mt-1.5 flex flex-wrap gap-1">
									{#each capabilitiesOf(row.plugin) as name (name)}
										<span class="badge badge-soft badge-xs">
											{labelOf(name)}{countOf(row.plugin, name) > 0
												? `: ${countOf(row.plugin, name)}`
												: ''}
										</span>
									{/each}
								</div>
							{/if}
						</div>

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

		{#if canSwitch}
			<h2 class="mt-10 text-sm font-semibold">To turn on</h2>

			{#if catalogLoading}
				<div class="mt-3 flex flex-col gap-2" aria-hidden="true">
					<span class="skeleton h-12 w-full"></span>
				</div>
			{:else if pluginState.status === 'error'}
				<div class="alert alert-error mt-3">
					<span>The hub answered no plugin to turn on. Reload the page.</span>
				</div>
			{:else if toTurnOn.length === 0}
				<div class="border-base-300 text-base-content/60 mt-3 rounded-md border border-dashed p-6">
					No other plugin is open to you. An administrator puts a plugin on your allowlist.
				</div>
			{:else}
				<ul class="border-base-300 rounded-box mt-3 divide-y border">
					{#each toTurnOn as plugin (plugin.id)}
						{@const name = displayNameOf(plugin, plugin.id)}
						<li class="flex items-center justify-between gap-3 px-4 py-3">
							<span class="min-w-0">
								<span class="block text-sm font-semibold">{name}</span>
								<span class="text-base-content/50 block truncate font-mono text-[11px]">
									{plugin.id}
								</span>
							</span>
							<input
								type="checkbox"
								class="toggle toggle-primary toggle-sm"
								aria-label="Use {name} in {data.workspace.name}"
								checked={false}
								disabled={busy !== null}
								onchange={(event) => toggle(plugin.id, event.currentTarget.checked)}
							/>
						</li>
					{/each}
				</ul>
			{/if}
		{/if}
	{/if}
</div>
