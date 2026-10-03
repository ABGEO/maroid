<script lang="ts">
	import { resolve } from '$app/paths';
	import { page } from '$app/state';

	import { workspaceById, workspaceState } from '$lib/state/workspaces.svelte';

	const current = $derived(workspaceById(page.params.workspace));
	const label = $derived(current?.name ?? 'Choose a workspace');
</script>

{#if workspaceState.status === 'ready' && workspaceState.workspaces.length > 0}
	<div class="dropdown">
		<div
			tabindex="0"
			role="button"
			class="btn btn-ghost btn-sm max-w-56 font-normal"
			aria-label="Switch workspace"
		>
			<span class="truncate">{label}</span>
			<svg
				xmlns="http://www.w3.org/2000/svg"
				width="12"
				height="12"
				viewBox="0 0 24 24"
				fill="none"
				stroke="currentColor"
				stroke-width="2"
				opacity="0.5"
			>
				<path d="m6 9 6 6 6-6" />
			</svg>
		</div>

		<ul
			tabindex="-1"
			class="dropdown-content menu menu-sm bg-base-100 border-base-300 rounded-box z-50 mt-2 w-60 border p-2 shadow-lg"
		>
			{#each workspaceState.workspaces as workspace (workspace.id)}
				<li>
					<a
						href={resolve('/(dashboard)/w/[workspace]', { workspace: workspace.id })}
						class:menu-active={workspace.id === current?.id}
					>
						<span class="truncate">{workspace.name}</span>
					</a>
				</li>
			{/each}
			<li class="border-base-300 mt-1 border-t pt-1">
				<a href={resolve('/workspaces/new')}>New workspace</a>
			</li>
		</ul>
	</div>
{/if}
