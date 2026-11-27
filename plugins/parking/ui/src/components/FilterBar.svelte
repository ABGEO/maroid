<script lang="ts">
	import type { Snippet } from 'svelte';

	let {
		filtered,
		onclear,
		children
	}: { filtered: boolean; onclear: () => void; children: Snippet } = $props();
</script>

<form
	class="panel mb-4 flex flex-wrap items-end gap-3 p-4"
	onsubmit={(event) => event.preventDefault()}
>
	{@render children()}
	{#if filtered}
		<button type="button" class="btn btn-sm btn-ghost" onclick={onclear}>Clear</button>
	{/if}
</form>
