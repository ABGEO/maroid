<script lang="ts">
	import { ApiError, PROBLEM_TYPE, api, isProblem, type EnabledPlugin } from '$lib/api';
	import { PERMISSION, holds } from '$lib/permissions';
	import { displayNameOf } from '$lib/plugins/capabilities';
	import { loadEnabled, pluginState } from '$lib/state/plugins.svelte';

	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

	const workspaceId = $derived(data.workspace.id);
	const canSwitch = $derived(holds(data.workspace, PERMISSION.pluginsWrite));

	let status = $state<'loading' | 'ready' | 'error'>('loading');
	let enabled = $state<EnabledPlugin[]>([]);
	let busy = $state<string | null>(null);
	let banner = $state<string | null>(null);

	/**
	 * The plugins that the person can switch on, then each enabled plugin that the list
	 * of the person lacks. An administrator can enable a plugin off the allowlist of a
	 * manager, and the manager still switches it off.
	 */
	const rows = $derived([
		...pluginState.plugins.map((plugin) => ({
			id: plugin.id,
			name: displayNameOf(plugin, plugin.id),
			listed: true
		})),
		...enabled
			.filter((one) => !pluginState.plugins.some((plugin) => plugin.id === one.plugin_id))
			.map((one) => ({
				id: one.plugin_id,
				name: displayNameOf(one.plugin, one.plugin_id),
				listed: false
			}))
	]);

	const enabledIds = $derived(enabled.map((one) => one.plugin_id));

	function failureMessage(error: unknown, fallback: string): string {
		if (error instanceof ApiError && isProblem(error.body)) {
			if (error.body.type === PROBLEM_TYPE.permissionDenied && error.body.detail) {
				return error.body.detail;
			}

			return error.body.errors?.[0]?.detail ?? error.body.title;
		}

		return fallback;
	}

	async function load(): Promise<void> {
		status = 'loading';

		try {
			const list = await api.enablements.list(workspaceId);
			if (list === null) {
				return;
			}

			enabled = list;
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
</script>

<div class="max-w-2xl">
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

	{#if status === 'loading' || pluginState.status === 'idle' || pluginState.status === 'loading'}
		<div class="mt-8 flex flex-col gap-2" aria-hidden="true">
			{#each [0, 1, 2] as i (i)}
				<span class="skeleton h-12 w-full"></span>
			{/each}
		</div>
	{:else if status === 'error' || pluginState.status === 'error'}
		<div class="alert alert-error mt-6">
			<span>The hub answered no plugin. Reload the page.</span>
		</div>
	{:else if rows.length === 0}
		<div class="border-base-300 text-base-content/60 mt-8 rounded-md border border-dashed p-6">
			No plugin is open to you. An administrator puts a plugin on your allowlist.
		</div>
	{:else}
		<ul class="border-base-300 rounded-box mt-8 divide-y border">
			{#each rows as row (row.id)}
				{@const on = enabledIds.includes(row.id)}
				<li class="flex items-center justify-between gap-3 px-4 py-3">
					<span class="min-w-0">
						<span class="block text-sm font-semibold">{row.name}</span>
						<span class="text-base-content/50 block truncate font-mono text-[11px]">{row.id}</span>
					</span>
					<input
						type="checkbox"
						class="toggle toggle-primary toggle-sm"
						aria-label="Use {row.name} in {data.workspace.name}"
						checked={on}
						disabled={!canSwitch || busy !== null || (!on && !row.listed)}
						onchange={(event) => toggle(row.id, event.currentTarget.checked)}
					/>
				</li>
			{/each}
		</ul>

		{#if !canSwitch}
			<p class="text-base-content/60 mt-3 text-sm">
				A manager of this workspace switches a plugin.
			</p>
		{/if}
	{/if}
</div>
