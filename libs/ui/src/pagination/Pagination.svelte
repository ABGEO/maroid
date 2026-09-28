<script lang="ts">
	import type { PageLinks, Pager } from './pager.svelte';

	let { pager, class: className = '' }: { pager: Pager<PageLinks>; class?: string } = $props();

	const loading = $derived(pager.status === 'loading');
</script>

{#if pager.hasPrevious || pager.hasNext}
	<nav class={['join grid grid-cols-2', className]} aria-label="Pagination">
		<button
			type="button"
			class="join-item btn btn-sm btn-outline"
			disabled={!pager.hasPrevious || loading}
			onclick={() => pager.previous()}
		>
			<span aria-hidden="true">&lt;</span> Previous
		</button>
		<button
			type="button"
			class="join-item btn btn-sm btn-outline"
			disabled={!pager.hasNext || loading}
			onclick={() => pager.next()}
		>
			Next <span aria-hidden="true">&gt;</span>
		</button>
	</nav>
{/if}
