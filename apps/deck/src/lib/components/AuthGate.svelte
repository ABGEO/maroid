<script lang="ts">
	import { onMount } from 'svelte';

	import { loadUser, userState } from '$lib/state/user.svelte';

	let { children } = $props();
	let loading = $state(true);

	onMount(async () => {
		await loadUser();

		if (userState.user) {
			loading = false;
		}
	});
</script>

{#if loading}
	<div class="flex min-h-screen items-center justify-center" role="status" aria-label="Loading">
		<span class="loading loading-spinner loading-xl text-primary"></span>
	</div>
{:else}
	{@render children()}
{/if}
