<script lang="ts">
	import { userState } from '$lib/state/user.svelte';

	let { children } = $props();
</script>

{#if userState.user?.is_administrator}
	{@render children()}
{:else}
	<div class="max-w-2xl">
		<header class="page-header">
			<div>
				<h1 class="page-title">Not found</h1>
				<p class="page-lead">This page does not exist.</p>
			</div>
		</header>
	</div>
{/if}
