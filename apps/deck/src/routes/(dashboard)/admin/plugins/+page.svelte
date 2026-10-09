<script lang="ts">
	import { resolve } from '$app/paths';

	import { api, type InstanceWorkspace } from '$lib/api';
	import { capabilitiesOf, countOf, displayNameOf, labelOf } from '$lib/plugins/capabilities';
	import { pluginState } from '$lib/state/plugins.svelte';

	let status = $state<'loading' | 'ready' | 'error'>('loading');
	let workspaces = $state<InstanceWorkspace[]>([]);

	function enabling(pluginId: string): InstanceWorkspace[] {
		return workspaces.filter((workspace) => workspace.plugin_ids.includes(pluginId));
	}

	async function load(): Promise<void> {
		status = 'loading';

		try {
			const list = await api.workspaces.listAll();
			if (list === null) {
				return;
			}

			workspaces = list;
			status = 'ready';
		} catch (error) {
			console.error('Failed to load the workspaces of the instance', error);
			status = 'error';
		}
	}

	$effect(() => {
		void load();
	});
</script>

<div class="max-w-3xl">
	<header class="page-header">
		<div>
			<h1 class="page-title">Plugins</h1>
			<p class="page-lead">
				Every plugin that the hub loaded, and the workspaces that use each one.
			</p>
		</div>
	</header>

	{#if status === 'loading' || pluginState.status === 'idle' || pluginState.status === 'loading'}
		<div class="flex flex-col gap-2" aria-hidden="true">
			{#each [0, 1, 2] as i (i)}
				<span class="skeleton h-16 w-full"></span>
			{/each}
		</div>
	{:else if status === 'error' || pluginState.status === 'error'}
		<div role="alert" class="alert alert-error alert-soft">
			<span>The hub answered no plugin. Reload the page.</span>
		</div>
	{:else if pluginState.plugins.length === 0}
		<div class="empty-state">The hub loaded no plugin.</div>
	{:else}
		<ul class="list panel">
			{#each pluginState.plugins as plugin (plugin.id)}
				{@const using = enabling(plugin.id)}
				<li class="list-row block">
					<div class="flex items-center gap-2">
						<span class="font-medium">{displayNameOf(plugin, plugin.id)}</span>
						<span class="badge badge-ghost badge-xs font-mono">v{plugin.version}</span>
					</div>
					<div class="meta truncate font-mono">{plugin.id}</div>
					{#if plugin.description}
						<p class="text-base-content/70 mt-1 text-sm">{plugin.description}</p>
					{/if}
					{#if capabilitiesOf(plugin).length > 0}
						<div class="mt-1.5 flex flex-wrap gap-1">
							{#each capabilitiesOf(plugin) as name (name)}
								<span class="badge badge-soft badge-xs">
									{labelOf(name)}{countOf(plugin, name) > 0 ? `: ${countOf(plugin, name)}` : ''}
								</span>
							{/each}
						</div>
					{/if}
					<div class="meta mt-2 flex flex-wrap items-center gap-1">
						{#if using.length === 0}
							No workspace uses it.
						{:else}
							<span>Used in</span>
							{#each using as workspace (workspace.id)}
								<a
									class="link link-hover"
									href={resolve('/(dashboard)/w/[workspace]/plugins', { workspace: workspace.id })}
								>
									{workspace.name}
								</a>
							{/each}
						{/if}
					</div>
				</li>
			{/each}
		</ul>
	{/if}
</div>
