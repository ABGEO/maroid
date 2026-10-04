<script lang="ts">
	import { resolve } from '$app/paths';

	import { api, type InstanceWorkspace } from '$lib/api';

	let status = $state<'loading' | 'ready' | 'error'>('loading');
	let workspaces = $state<InstanceWorkspace[]>([]);

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
	<h1 class="font-display text-[42px] leading-[1.05] tracking-tight">Workspaces</h1>
	<p class="text-base-content/60 mt-3 text-sm">
		Every workspace of this instance. You manage its members and its plugins, and you read none of
		its records.
	</p>

	{#if status === 'loading'}
		<div class="mt-8 flex flex-col gap-2" aria-hidden="true">
			{#each [0, 1, 2] as i (i)}
				<span class="skeleton h-12 w-full"></span>
			{/each}
		</div>
	{:else if status === 'error'}
		<div class="alert alert-error mt-6">
			<span>The hub answered no workspace. Reload the page.</span>
		</div>
	{:else if workspaces.length === 0}
		<div class="border-base-300 text-base-content/60 mt-8 rounded-md border border-dashed p-6">
			This instance holds no workspace.
		</div>
	{:else}
		<ul class="border-base-300 rounded-box mt-8 divide-y border">
			{#each workspaces as workspace (workspace.id)}
				<li class="flex flex-wrap items-center justify-between gap-3 px-4 py-3">
					<span class="min-w-0">
						<span class="block text-sm font-semibold">{workspace.name}</span>
						<span class="text-base-content/50 block text-[12px]">
							{workspace.member_count}
							{workspace.member_count === 1 ? 'member' : 'members'} ·
							{workspace.plugin_ids.length === 0 ? 'no plugin' : workspace.plugin_ids.join(', ')}
						</span>
					</span>
					<span class="flex gap-1">
						<a
							class="btn btn-ghost btn-xs"
							href={resolve('/(dashboard)/w/[workspace]/members', { workspace: workspace.id })}
						>
							Members
						</a>
						<a
							class="btn btn-ghost btn-xs"
							href={resolve('/(dashboard)/w/[workspace]/plugins', { workspace: workspace.id })}
						>
							Plugins
						</a>
					</span>
				</li>
			{/each}
		</ul>
	{/if}
</div>
