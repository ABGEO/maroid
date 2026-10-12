<script lang="ts">
	import { resolve } from '$app/paths';

	import { api } from '$lib/api';
	import Alert from '$lib/components/ui/Alert.svelte';

	import WidgetCard from './WidgetCard.svelte';
	import { GRID_COLUMNS, spanOf } from './layout';

	interface Props {
		workspaceId: string;
	}

	let { workspaceId }: Props = $props();

	const widgets = $derived(api.widgets.list(workspaceId));
</script>

{#snippet placeholder()}
	<div class={['grid auto-rows-[10rem] gap-4', GRID_COLUMNS]} aria-hidden="true">
		<span class={['skeleton', spanOf('s')]}></span>
		<span class={['skeleton', spanOf('m')]}></span>
	</div>
{/snippet}

{#await widgets}
	{@render placeholder()}
{:then listed}
	{#if listed === null}
		{@render placeholder()}
	{:else if listed.length === 0}
		<div class="empty-state">
			<p>No plugin of this workspace shows a widget yet.</p>
			<a
				class="link link-primary mt-2 inline-block"
				href={resolve('/(dashboard)/w/[workspace]/plugins', { workspace: workspaceId })}
			>
				Open the plugins
			</a>
		</div>
	{:else}
		<section class={['grid auto-rows-[10rem] gap-4', GRID_COLUMNS]} aria-label="Widgets">
			{#each listed as widget (`${widget.plugin_id}/${widget.key}`)}
				<WidgetCard {workspaceId} {widget} />
			{/each}
		</section>
	{/if}
{:catch}
	<Alert kind="error">
		<span>The widgets did not load.</span>
	</Alert>
{/await}
