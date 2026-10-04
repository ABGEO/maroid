<script lang="ts">
	import { isMember, rememberWorkspace, workspaceState } from '$lib/state/workspaces.svelte';

	import type { LayoutProps } from './$types';

	let { data, children }: LayoutProps = $props();

	const managing = $derived(workspaceState.status === 'ready' && !isMember(data.workspace.id));

	$effect(() => {
		if (!managing) {
			rememberWorkspace(data.workspace.id);
		}
	});
</script>

{#if managing}
	<div class="alert alert-info mb-6 max-w-2xl">
		<span class="text-sm">
			You manage {data.workspace.name} as an administrator. You change its members and its plugins, and
			you read none of its records.
		</span>
	</div>
{/if}

{@render children()}
