<script lang="ts">
	import favicon from '$lib/assets/favicon.svg';

	import '../app.css';

	let { children } = $props();
</script>

<svelte:head><link rel="icon" href={favicon} /></svelte:head>

<main class="bg-base-100 min-h-screen">
	{@render children()}
</main>
