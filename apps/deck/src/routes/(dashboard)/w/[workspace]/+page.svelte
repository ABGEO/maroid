<script lang="ts">
	import WidgetGrid from '$lib/components/widgets/WidgetGrid.svelte';

	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
</script>

<div class="max-w-5xl">
	<header class="page-header">
		<div>
			<h1 class="page-title">{data.workspace.name}</h1>
			<p class="page-lead">The current state of the plugins of this workspace.</p>
		</div>
	</header>

	<WidgetGrid workspaceId={data.workspace.id} />
</div>
