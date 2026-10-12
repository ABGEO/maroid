<script lang="ts">
	import { api } from '$lib/api';
	import type { Widget } from '$lib/api/types';

	import ListWidget from './ListWidget.svelte';
	import StatWidget from './StatWidget.svelte';
	import { linkOf, spanOf } from './layout';

	interface Props {
		workspaceId: string;
		widget: Widget;
	}

	let { workspaceId, widget }: Props = $props();

	const answer = $derived(api.widgets.read(workspaceId, widget.plugin_id, widget.key));

	function href(path: string): string {
		return linkOf(workspaceId, widget.plugin_id, path);
	}
</script>

<article class={['panel flex min-h-0 flex-col p-4', spanOf(widget.size)]}>
	<h2 class="section-title mb-2 truncate">{widget.title}</h2>

	{#await answer}
		<div class="flex flex-1 flex-col gap-2" role="status" aria-label="Loading">
			<span class="skeleton h-4 w-3/4"></span>
			<span class="skeleton h-4 w-1/2"></span>
		</div>
	{:then result}
		{#if result.state === 'ready' && result.stat}
			<StatWidget stat={result.stat} {href} />
		{:else if result.state === 'ready' && result.list}
			<ListWidget list={result.list} size={widget.size} {href} />
		{:else if result.state === 'empty'}
			<p class="meta">{result.message}</p>
		{:else if result.state === 'not_set_up' && result.link}
			<p class="text-sm">{result.message}</p>
			<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- linkOf resolves every target -->
			<a class="link link-primary mt-auto text-sm" href={href(result.link)}>Set up</a>
		{/if}
	{:catch}
		<p class="text-error text-sm">This widget did not load.</p>
	{/await}
</article>
