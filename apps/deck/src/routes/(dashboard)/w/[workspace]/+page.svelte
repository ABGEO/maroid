<script lang="ts">
	import WidgetGrid from '$lib/components/widgets/WidgetGrid.svelte';
	import { isMember, workspaceState } from '$lib/state/workspaces.svelte';

	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

	// An administrator manages a workspace of another person and reads none of its
	// records, and the layout tells them so.
	const member = $derived(workspaceState.status === 'ready' && isMember(data.workspace.id));
</script>

<div class="max-w-5xl">
	<header class="page-header">
		<div>
			<h1 class="page-title">{data.workspace.name}</h1>
			<p class="page-lead">The current state of the plugins of this workspace.</p>
		</div>
	</header>

	{#if member}
		<WidgetGrid workspaceId={data.workspace.id} />
	{/if}
</div>
