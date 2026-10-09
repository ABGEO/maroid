<script lang="ts">
	import { resolve } from '$app/paths';

	import { api, type InstanceWorkspace } from '$lib/api';
	import PluginSummary from '$lib/components/plugins/PluginSummary.svelte';
	import Alert from '$lib/components/ui/Alert.svelte';
	import SkeletonList from '$lib/components/ui/SkeletonList.svelte';
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
		<SkeletonList row="h-16" />
	{:else if status === 'error' || pluginState.status === 'error'}
		<Alert kind="error"><span>The hub answered no plugin. Reload the page.</span></Alert>
	{:else if pluginState.plugins.length === 0}
		<div class="empty-state">The hub loaded no plugin.</div>
	{:else}
		<ul class="list panel">
			{#each pluginState.plugins as plugin (plugin.id)}
				{@const using = enabling(plugin.id)}
				<li class="list-row block">
					<PluginSummary id={plugin.id} {plugin} detailed>
						<div class="meta mt-2 flex flex-wrap items-center gap-1">
							{#if using.length === 0}
								No workspace uses it.
							{:else}
								<span>Used in</span>
								{#each using as workspace (workspace.id)}
									<a
										class="link link-hover"
										href={resolve('/(dashboard)/w/[workspace]/plugins', {
											workspace: workspace.id
										})}
									>
										{workspace.name}
									</a>
								{/each}
							{/if}
						</div>
					</PluginSummary>
				</li>
			{/each}
		</ul>
	{/if}
</div>
