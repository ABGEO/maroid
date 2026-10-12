<script lang="ts">
	import type { Snippet } from 'svelte';

	import { api } from '$lib/api';
	import type { Widget } from '$lib/api/types';
	import SkeletonList from '$lib/components/ui/SkeletonList.svelte';

	import ListWidget from './ListWidget.svelte';
	import StatWidget from './StatWidget.svelte';
	import { linkOf, spanOf } from './layout';

	interface Props {
		workspaceId: string;
		widget: Widget;
	}

	let { workspaceId, widget }: Props = $props();

	const titleId = $props.id();
	const answer = $derived(api.widgets.read(workspaceId, widget.plugin_id, widget.key));

	function href(path: string): string {
		return linkOf(workspaceId, widget.plugin_id, path);
	}
</script>

{#snippet card(busy: boolean, body: Snippet)}
	<article
		class={['panel flex min-h-0 flex-col p-4', spanOf(widget.size)]}
		aria-labelledby={titleId}
		aria-busy={busy}
	>
		<h2 id={titleId} class="section-title mb-2 truncate">{widget.title}</h2>
		{@render body()}
	</article>
{/snippet}

{#snippet loading()}
	<SkeletonList count={2} row="h-4" />
{/snippet}

{#snippet failed()}
	<p class="text-error text-sm">This widget did not load.</p>
{/snippet}

{#await answer}
	{@render card(true, loading)}
{:then result}
	{#snippet shown()}
		{#if result === null}
			{@render loading()}
		{:else if result.state === 'ready' && widget.kind === 'stat' && result.stat}
			<StatWidget stat={result.stat} {href} />
		{:else if result.state === 'ready' && widget.kind === 'list' && result.list}
			<ListWidget list={result.list} size={widget.size} {href} />
		{:else if result.state === 'empty'}
			<p class="meta">{result.message}</p>
		{:else if result.state === 'not_set_up' && result.link}
			<p class="text-sm">{result.message}</p>
			<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- linkOf resolves every target -->
			<a class="link link-primary mt-auto text-sm" href={href(result.link)}>Set up</a>
		{/if}
	{/snippet}
	{@render card(result === null, shown)}
{:catch}
	{@render card(false, failed)}
{/await}
