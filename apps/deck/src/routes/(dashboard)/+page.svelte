<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';

	import CreateWorkspaceForm from '$lib/components/workspace/CreateWorkspaceForm.svelte';
	import { landingWorkspace, workspaceState } from '$lib/state/workspaces.svelte';

	const landing = $derived(
		workspaceState.status === 'ready' ? landingWorkspace(workspaceState.workspaces) : null
	);

	$effect(() => {
		if (landing) {
			void goto(resolve('/(dashboard)/w/[workspace]', { workspace: landing.id }), {
				replaceState: true
			});
		}
	});
</script>

{#if workspaceState.status === 'ready' && workspaceState.workspaces.length === 0}
	<div class="max-w-2xl">
		<header class="page-header">
			<div>
				<h1 class="page-title">Create a workspace</h1>
				<p class="page-lead">
					A workspace holds the records of your plugins. Keep it to yourself, or add the people you
					share it with.
				</p>
			</div>
		</header>

		<CreateWorkspaceForm />
	</div>
{:else if workspaceState.status === 'error'}
	<div role="alert" class="alert alert-error alert-soft max-w-2xl">
		<span>The hub answered no workspace. Reload the page.</span>
	</div>
{:else}
	<div class="flex items-center justify-center py-20" role="status" aria-label="Loading">
		<span class="loading loading-spinner loading-lg text-primary"></span>
	</div>
{/if}
