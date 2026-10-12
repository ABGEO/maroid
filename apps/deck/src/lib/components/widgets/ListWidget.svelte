<script lang="ts">
	import type { ListData, WidgetSize } from '$lib/api/types';

	import { itemsOf } from './layout';

	interface Props {
		list: ListData;
		size: WidgetSize;
		href: (path: string) => string;
	}

	let { list, size, href }: Props = $props();

	const shown = $derived(list.items.slice(0, itemsOf(size)));
	const hidden = $derived(list.items.length - shown.length);
</script>

{#snippet text(label: string, meta: string | undefined)}
	<span class="list-col-grow min-w-0">
		<span class="block truncate text-sm font-medium">{label}</span>
		{#if meta}
			<span class="meta block truncate">{meta}</span>
		{/if}
	</span>
{/snippet}

<ul class="list -mx-2 min-h-0 flex-1 overflow-hidden">
	{#each shown as item, index (index)}
		<li>
			{#if item.link}
				<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- linkOf resolves every target -->
				<a class="list-row hover:bg-base-200 items-center px-2 py-1" href={href(item.link)}>
					{@render text(item.label, item.meta)}
					<span class="text-base-content/40" aria-hidden="true">&rarr;</span>
				</a>
			{:else}
				<div class="list-row items-center px-2 py-1">
					{@render text(item.label, item.meta)}
				</div>
			{/if}
		</li>
	{/each}
</ul>

{#if hidden > 0 || list.more}
	<div class="mt-2 flex items-center justify-between gap-2 text-sm">
		{#if hidden > 0}
			<span class="meta">{hidden} more</span>
		{/if}
		{#if list.more}
			<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- linkOf resolves every target -->
			<a class="link link-primary ms-auto" href={href(list.more)}>See all</a>
		{/if}
	</div>
{/if}
