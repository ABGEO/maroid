<script lang="ts">
	import { userState } from '$lib/state/user.svelte';

	let { children } = $props();
</script>

{#if userState.user?.is_administrator}
	{@render children()}
{:else}
	<div class="max-w-2xl">
		<h1 class="font-display text-[42px] leading-[1.05] tracking-tight">Not found</h1>
		<p class="text-base-content/60 mt-3 text-sm">This page does not exist.</p>
	</div>
{/if}
