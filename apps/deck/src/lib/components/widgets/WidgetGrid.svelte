<script lang="ts">
	import { api } from '$lib/api';
	import Alert from '$lib/components/ui/Alert.svelte';

	import WidgetCard from './WidgetCard.svelte';
	import { GRID_COLUMNS } from './layout';

	interface Props {
		workspaceId: string;
	}

	let { workspaceId }: Props = $props();

	const widgets = $derived(api.widgets.list(workspaceId));
</script>

{#await widgets then listed}
	{#if listed && listed.length > 0}
		<section class={['mb-8 grid auto-rows-[10rem] gap-4', GRID_COLUMNS]} aria-label="Widgets">
			{#each listed as widget (`${widget.plugin_id}/${widget.key}`)}
				<WidgetCard {workspaceId} {widget} />
			{/each}
		</section>
	{/if}
{:catch}
	<Alert kind="error" class="mb-8">
		<span>The widgets did not load.</span>
	</Alert>
{/await}
