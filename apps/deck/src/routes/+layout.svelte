<script lang="ts">
	import '../app.css';

	let { children } = $props();
</script>

<main class="bg-base-100 min-h-screen">
	{@render children()}
</main>
