<script lang="ts">
	import { resolve } from '$app/paths';

	import { api, type InstanceWorkspace } from '$lib/api';
	import Alert from '$lib/components/ui/Alert.svelte';
	import SkeletonList from '$lib/components/ui/SkeletonList.svelte';
	import RemoveWorkspaceDialog from '$lib/components/workspace/RemoveWorkspaceDialog.svelte';
	import { loadWorkspaces } from '$lib/state/workspaces.svelte';

	let status = $state<'loading' | 'ready' | 'error'>('loading');
	let workspaces = $state<InstanceWorkspace[]>([]);
	let removal = $state<ReturnType<typeof RemoveWorkspaceDialog>>();

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

	async function removed(): Promise<void> {
		await Promise.all([load(), loadWorkspaces()]);
	}
</script>

<div class="max-w-3xl">
	<header class="page-header">
		<div>
			<h1 class="page-title">Workspaces</h1>
			<p class="page-lead">
				Every workspace of this instance. You manage its members and its plugins, and you read none
				of its records.
			</p>
		</div>
	</header>

	{#if status === 'loading'}
		<SkeletonList row="h-16" />
	{:else if status === 'error'}
		<Alert kind="error"><span>The workspaces did not load. Reload the page.</span></Alert>
	{:else if workspaces.length === 0}
		<div class="empty-state">This instance holds no workspace.</div>
	{:else}
		<ul class="list panel">
			{#each workspaces as workspace (workspace.id)}
				<li class="list-row items-center">
					<span class="list-col-grow min-w-0">
						<span class="block font-medium">{workspace.name}</span>
						<span class="meta block truncate">
							{workspace.member_count}
							{workspace.member_count === 1 ? 'member' : 'members'} ·
							{workspace.plugin_ids.length === 0 ? 'no plugin' : workspace.plugin_ids.join(', ')}
						</span>
					</span>
					<a
						class="btn btn-ghost btn-sm"
						href={resolve('/(dashboard)/w/[workspace]/members', { workspace: workspace.id })}
					>
						Members
					</a>
					<a
						class="btn btn-ghost btn-sm"
						href={resolve('/(dashboard)/w/[workspace]/plugins', { workspace: workspace.id })}
					>
						Plugins
					</a>
					<button
						type="button"
						class="btn btn-ghost btn-sm text-error"
						onclick={() => removal?.open(workspace)}
					>
						Remove
					</button>
				</li>
			{/each}
		</ul>
	{/if}
</div>

<RemoveWorkspaceDialog bind:this={removal} onremoved={removed} />
