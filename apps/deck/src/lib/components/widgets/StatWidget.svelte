<script lang="ts">
	import type { StatData } from '$lib/api/types';

	interface Props {
		stat: StatData;
		href: (path: string) => string;
	}

	let { stat, href }: Props = $props();
</script>

{#snippet figure()}
	<span class="min-w-0 flex-1">
		<span class="flex items-baseline gap-1.5">
			<span class="truncate text-3xl font-semibold tabular-nums">{stat.value}</span>
			{#if stat.unit}
				<span class="meta text-sm">{stat.unit}</span>
			{/if}
		</span>
		{#if stat.caption}
			<span class="meta mt-1 block truncate">{stat.caption}</span>
		{/if}
	</span>
{/snippet}

{#if stat.link}
	<!-- eslint-disable svelte/no-navigation-without-resolve -- linkOf resolves every target -->
	<a
		class="rounded-box hover:bg-base-200 -mx-2 flex flex-1 items-center gap-2 px-2"
		href={href(stat.link)}
	>
		{@render figure()}
		<span class="text-base-content/40" aria-hidden="true">&rarr;</span>
	</a>
	<!-- eslint-enable svelte/no-navigation-without-resolve -->
{:else}
	<div class="flex flex-1 items-center">
		{@render figure()}
	</div>
{/if}
